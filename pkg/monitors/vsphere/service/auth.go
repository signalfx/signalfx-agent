package service

import (
	"context"
	"crypto/tls"
	"net/url"

	"github.com/signalfx/signalfx-agent/pkg/monitors/vsphere/model"
	"github.com/sirupsen/logrus"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/session"
	"github.com/vmware/govmomi/vim25"
	"github.com/vmware/govmomi/vim25/soap"
)

type AuthService struct {
	log *logrus.Entry
}

func NewAuthService(log *logrus.Entry) *AuthService {
	return &AuthService{log: log}
}

// LogIn logs into vCenter and returns a logged-in Client or an error
func (svc *AuthService) LogIn(ctx context.Context, conf *model.Config) (*govmomi.Client, error) {
	myURL, err := url.Parse("https://" + conf.Host + "/sdk")
	if err != nil {
		return nil, err
	}
	myURL.User = url.UserPassword(conf.Username, conf.Password)

	svc.log.WithFields(logrus.Fields{
		"ip":   conf.Host,
		"user": conf.Username,
	}).Info("Connecting to vsphereInfo")

	client, err := svc.newGovmomiClient(ctx, myURL, conf)
	if err != nil {
		return nil, err
	}

	err = client.Login(ctx, myURL.User)
	if err != nil {
		return nil, err
	}

	return client, nil
}

func (svc *AuthService) newGovmomiClient(ctx context.Context, myURL *url.URL, conf *model.Config) (*govmomi.Client, error) {
	vimClient, err := svc.newVimClient(ctx, myURL, conf)
	if err != nil {
		return nil, err
	}
	return &govmomi.Client{
		Client:         vimClient,
		SessionManager: session.NewManager(vimClient),
	}, nil
}

func (svc *AuthService) newVimClient(ctx context.Context, myURL *url.URL, conf *model.Config) (*vim25.Client, error) {
	soapClient := soap.NewClient(myURL, conf.InsecureSkipVerify)
	if conf.TLSCACertPath != "" {
		svc.log.Info("Attempting to load TLSCACertPath from ", conf.TLSCACertPath)
		err := soapClient.SetRootCAs(conf.TLSCACertPath)
		if err != nil {
			return nil, err
		}
	} else {
		svc.log.Info("No tlsCACertPath provided. Not setting root CA.")
	}
	if conf.TLSClientCertificatePath != "" && conf.TLSClientKeyPath != "" {
		svc.log.Infof(
			"Attempting to load client certificate from TLSClientCertificatePath(%s) and TLSClientKeyPath(%s)",
			conf.TLSClientCertificatePath,
			conf.TLSClientKeyPath,
		)
		cert, err := tls.LoadX509KeyPair(conf.TLSClientCertificatePath, conf.TLSClientKeyPath)
		if err != nil {
			return nil, err
		}
		soapClient.SetCertificate(cert)
	} else {
		svc.log.Info("Either or both of tlsClientCertificatePath or tlsClientKeyPath not provided. Not setting client certificate.")
	}
	return vim25.NewClient(ctx, soapClient)
}
