package service_provider

import (
	"testing"
	"tinvest/internal/config"
)

func TestDensityDisabledDoesNotCreateClients(t *testing.T) {
	s := &ServiceProvider{appConfig: &config.Config{Density: config.NewDensityConfig()}}
	got, err := s.GetDensityService()
	if err != nil || got != nil {
		t.Fatalf("service=%v err=%v", got, err)
	}
}
func TestDensityEnabledRequiresDestination(t *testing.T) {
	c := config.NewDensityConfig()
	c.Enabled = true
	s := &ServiceProvider{appConfig: &config.Config{Density: c}}
	if _, err := s.GetDensityService(); err == nil {
		t.Fatal("missing destination accepted")
	}
}
