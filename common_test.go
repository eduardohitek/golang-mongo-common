package common

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func Test_setClientOptions(t *testing.T) {
	connectionURI := "mongodb://localhost"
	appName := "common-test"
	client := setClientOptions(connectionURI, appName)
	assert.Equal(t, appName, *client.AppName)
	assert.Equal(t, ConnectTimeout, *client.ConnectTimeout)
	assert.Equal(t, MaxConnIdleTime, *client.MaxConnIdleTime)
	assert.Equal(t, ServerSelectionTimeout, *client.ServerSelectionTimeout)
	assert.Equal(t, connectionURI, client.GetURI())
}

func Test_setClientOptionsWithCredentials(t *testing.T) {
	connectionURI := "mongodb://localhost"
	appName := "common-test"
	authDB := "admin"
	userDB := "user"
	passDB := "password"
	credentials := options.Credential{AuthSource: authDB, Username: userDB, Password: passDB}
	client := setClientOptionsWithCredentials(connectionURI, appName, credentials)
	assert.Equal(t, appName, *client.AppName)
	assert.Equal(t, ConnectTimeout, *client.ConnectTimeout)
	assert.Equal(t, MaxConnIdleTime, *client.MaxConnIdleTime)
	assert.Equal(t, ServerSelectionTimeout, *client.ServerSelectionTimeout)
	assert.Equal(t, connectionURI, client.GetURI())
	assert.Equal(t, authDB, client.Auth.AuthSource)
	assert.Equal(t, userDB, client.Auth.Username)
	assert.Equal(t, passDB, client.Auth.Password)
}

func Test_returnClient(t *testing.T) {
	connectionURI := "localhost"
	appName := "common-test"
	client, err := ReturnClient(connectionURI, appName)
	assert.Nil(t, err, "should be nil")
	assert.NotNil(t, client, "should not be nil")
	err = client.Ping(context.TODO(), nil)
	assert.Nil(t, err, "should be nil")

}
