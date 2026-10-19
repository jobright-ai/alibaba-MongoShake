package utils

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func TestEncodeMongoURIQueryOptions(t *testing.T) {
	for _, uri := range []string{
		"mongodb://user:password@host1:27017,host2:27017/?authSource=admin",
		"mongodb://user:password@host1:27017/app?authSource=admin&replicaSet=rs0",
		"mongodb://user:password@host1:27017/?readPreference=secondaryPreferred&directConnection=true",
	} {
		t.Run(uri, func(t *testing.T) {
			encoded, err := EncodeMongoURI(uri)
			require.NoError(t, err)
			require.Equal(t, uri, encoded, "URI options must remain query parameters")
			require.NoError(t, options.Client().ApplyURI(encoded).Validate())
		})
	}

	encoded, err := EncodeMongoURI("mongodb://user:p@ss!@host1:27017/?authSource=admin")
	require.NoError(t, err)
	require.Equal(t, "mongodb://user:p%40ss%21@host1:27017/?authSource=admin", encoded)
	require.NoError(t, options.Client().ApplyURI(encoded).Validate())
}

func TestGetDBVersionRejectsUninitializedConnection(t *testing.T) {
	for _, conn := range []*MongoCommunityConn{nil, {}} {
		version, err := GetDBVersion(conn)
		require.Error(t, err)
		require.Empty(t, version)
	}
}
