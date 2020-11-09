package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInvFilterMatch(t *testing.T) {
	f, err := NewFilter("Datacenter == 'dc0' && Cluster == 'cluster0'")
	require.NoError(t, err)
	dims := pairs{
		pair{dimDatacenter, "dc0"},
		pair{dimCluster, "cluster0"},
	}
	follow, err := f.shouldFollowCluster(dims)
	require.NoError(t, err)
	require.True(t, follow)
}

func TestInvFilterNoMatch(t *testing.T) {
	f, err := NewFilter("Datacenter == 'dc0' && Cluster == 'xyz'")
	require.NoError(t, err)
	dims := pairs{
		pair{dimDatacenter, "dc0"},
		pair{dimCluster, "cluster0"},
	}
	follow, err := f.shouldFollowCluster(dims)
	require.NoError(t, err)
	require.False(t, follow)
}

func TestInvFilterDCMatch(t *testing.T) {
	f, err := NewFilter("Datacenter == 'dc0'")
	require.NoError(t, err)
	dims := pairs{
		pair{dimDatacenter, "dc0"},
		pair{dimCluster, "cluster0"},
	}
	follow, err := f.shouldFollowCluster(dims)
	require.NoError(t, err)
	require.True(t, follow)
}

func TestInvFilterClusterMatch(t *testing.T) {
	f, err := NewFilter("Cluster == 'cluster0'")
	require.NoError(t, err)
	dims := pairs{
		pair{dimDatacenter, "dc0"},
		pair{dimCluster, "cluster0"},
	}
	follow, err := f.shouldFollowCluster(dims)
	require.NoError(t, err)
	require.True(t, follow)
}
