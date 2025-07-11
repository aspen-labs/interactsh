package server

import (
	"testing"

	"github.com/projectdiscovery/interactsh/pkg/settings"
	"github.com/stretchr/testify/require"
)

func TestGetURLIDComponent(t *testing.T) {
	options := Options{CorrelationIdLength: settings.CorrelationIdLengthDefault, CorrelationIdNonceLength: settings.CorrelationIdNonceLengthDefault}
	random := options.getURLIDComponent("c6rj61aciaeutn2ae680cg5.ugboyyyyyn.interactsh.com")
	require.Equal(t, "ugboyyyyyn", random, "could not get correct component")
}
