package prometheus

import (
	"testing"
	"time"

	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/stretchr/testify/assert"
)

func TestStore(t *testing.T) {
	// given
	alertKey := "12"
	alertValue := "1.23"

	// when
	store := NewAlertCache(AlertCacheConfig{TTL: 2})
	store.Put(alertKey, v1.Alert{Value: alertValue})
	alert := store.Get(alertKey)

	//then
	assert.NotNil(t, alert)
	assert.Equal(t, alertValue, alert.Value)
	time.Sleep(4 * time.Second)
	alert2 := store.Get(alertKey)
	assert.Nil(t, alert2)
}
