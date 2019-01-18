// Package vault contains the logic for using Vault as a remote config source
//
// How to use auth methods with Vault Go client: https://groups.google.com/forum/#!msg/vault-tool/cS7J2KbAwZg/7pu6PYSRAAAJ
package vault

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/signalfx/signalfx-agent/internal/core/config/types"
	log "github.com/sirupsen/logrus"

	"github.com/hashicorp/vault/api"
)

var logger = log.WithFields(log.Fields{"remoteConfigSource": "vault"})

type vaultConfigSource struct {
	// The Vault client
	client *api.Client
	// Secrets that have been read from Vault
	secretsByVaultPath                map[string]*api.Secret
	renewersByVaultPath               map[string]*api.Renewer
	nonRenewableVaultPathRefetchTimes map[string]time.Time
	// Used for unit testing
	nowProvider  func() time.Time
	conf         *Config
	tokenRenewer *api.Renewer
}

var _ types.Stoppable = &vaultConfigSource{}

// Config for the vault remote config
type Config struct {
	// The Vault Address.  Can also be provided by the standard Vault envvar
	// `VAULT_ADDR`.  This option takes priority over the envvar if provided.
	VaultAddr string `yaml:"vaultAddr"`
	// The Vault token, can also be provided by it the standard Vault envvar
	// `VAULT_TOKEN`.  This option takes priority over the envvar if provided.
	VaultToken string `yaml:"vaultToken" neverLog:"true"`
}

// Validate the config
func (c *Config) Validate() error {
	if c.VaultToken == "" {
		if os.Getenv("VAULT_TOKEN") == "" {
			return errors.New("vault token is required, either in the agent config or the envvar VAULT_TOKEN")
		}

		c.VaultToken = os.Getenv("VAULT_TOKEN")
	}
	return nil
}

var _ types.ConfigSourceConfig = &Config{}

// New creates a new Vault remote config source from the target config
func (c *Config) New() (types.ConfigSource, error) {
	return New(c)
}

// New creates a new vault ConfigSource
func New(conf *Config) (types.ConfigSource, error) {
	logger.Info("Initializing new Vault remote config instance")

	c, err := api.NewClient(&api.Config{
		Address: conf.VaultAddr,
	})
	if err != nil {
		return nil, err
	}

	c.SetToken(conf.VaultToken)

	vcs := &vaultConfigSource{
		client:                            c,
		secretsByVaultPath:                make(map[string]*api.Secret),
		renewersByVaultPath:               make(map[string]*api.Renewer),
		nonRenewableVaultPathRefetchTimes: make(map[string]time.Time),
		nowProvider:                       time.Now,
		conf:                              conf,
	}

	// This will change if we ever support auth methods for getting the token
	vcs.initTokenRenewalIfNeeded()

	return vcs, nil
}

func (v *vaultConfigSource) Name() string {
	return "vault"
}

func (v *vaultConfigSource) Get(path string) (map[string][]byte, uint64, error) {
	vaultPath, key, err := splitConfigPath(path)
	if err != nil {
		return nil, 0, err
	}

	secret, ok := v.secretsByVaultPath[vaultPath]
	if !ok {
		logger.Debugf("Reading Vault secret at path: %s", vaultPath)

		secret, err = v.client.Logical().Read(vaultPath)
		if err != nil {
			return nil, 0, err
		}

		if secret == nil {
			return nil, 0, fmt.Errorf("no secret found at path %s", vaultPath)
		}

		if secret.Renewable {
			renewer, err := v.client.NewRenewer(&api.RenewerInput{
				Secret: secret,
			})
			if err == nil {
				logger.Debugf("Setting up Vault renewer for secret at path %s", vaultPath)
				v.renewersByVaultPath[vaultPath] = renewer
				go renewer.Renew()
			} else {
				logger.Errorf("Could not set up renewal on Vault secret at path %s: %v", vaultPath, err)
			}
		} else if secret.LeaseDuration > 0 {
			// We have a secret that isn't renewable but still expires.  We
			// need to just refetch it before it expires.  Set the refetch time
			// to half the lease duration.
			logger.Debugf("Secret at path %s cannot be renewed, refetching within %d seconds", vaultPath, secret.LeaseDuration)
			v.nonRenewableVaultPathRefetchTimes[vaultPath] = time.Now().Add(time.Duration(secret.LeaseDuration/2) * time.Second)
		}
		v.secretsByVaultPath[vaultPath] = secret
	}

	for _, w := range secret.Warnings {
		logger.Warnf("Warning received for Vault secret at path %s: %s", vaultPath, w)
	}

	if val := traverseToKey(secret.Data, key); val != nil {
		logger.Debugf("Fetched secret at %s -> %s", vaultPath, key)

		return map[string][]byte{
			path: []byte(fmt.Sprintf("%#v", val)),
		}, 0, nil
	}

	return nil, 0, fmt.Errorf("no key %s found in Vault secret %s", key, vaultPath)
}

// Vault doesn't have a "watch" concept but we do have to renew tokens, so
// watch for errors doing that.
func (v *vaultConfigSource) WaitForChange(path string, version uint64, stop <-chan struct{}) error {
	vaultPath, _, err := splitConfigPath(path)
	if err != nil {
		return err
	}
	renewer := v.renewersByVaultPath[vaultPath]

	var watchErr error

	if renewer == nil {
		refetchTime := v.nonRenewableVaultPathRefetchTimes[vaultPath]
		if refetchTime.IsZero() {
			select {
			case <-stop:
				break
			}
		} else {
			timer := time.NewTimer(time.Until(refetchTime))
			defer timer.Stop()
			select {
			case <-stop:
				break
			case <-timer.C:
				break
			}
		}
	} else {
		select {
		// This will receive if there are an errors renewing a secret lease
		case watchErr = <-renewer.DoneCh():
			break
		case <-stop:
			renewer.Stop()
		}
	}

	// Wipe the secret from the cache so that it gets refetched
	delete(v.renewersByVaultPath, vaultPath)
	delete(v.secretsByVaultPath, vaultPath)
	delete(v.nonRenewableVaultPathRefetchTimes, vaultPath)

	logger.Debugf("Path changed or failed to renew: %s", vaultPath)

	return watchErr
}

func (v *vaultConfigSource) Stop() error {
	if v.tokenRenewer != nil {
		v.tokenRenewer.Stop()
	}
	return nil
}
