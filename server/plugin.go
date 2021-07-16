package main

import (
	"encoders"
	"net/http"
	"sync"
	"time"
	"utils"

	"github.com/gorilla/mux"
	"github.com/mattermost/mattermost-server/v5/plugin"
	"github.com/patrickmn/go-cache"
)

type Plugin struct {
	plugin.MattermostPlugin
	router            *mux.Router
	globalCache       *cache.Cache
	internalKey       []byte
	encoder           encoders.Encoder
	configurationLock sync.RWMutex
	configuration     *configuration
}

func (p *Plugin) OnActivate() error {
	p.router = p.forkRouter()
	p.internalKey = []byte(utils.GenerateKey())
	p.globalCache = cache.New(5*time.Minute, 5*time.Minute)
	return nil
}

func (p *Plugin) ServeHTTP(c *plugin.Context, w http.ResponseWriter, r *http.Request) {
	p.router.ServeHTTP(w, r)
}
