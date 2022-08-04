package http

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"reflect"
	"strings"
	"sync"
	"time"

	ginzap "github.com/gin-contrib/zap"
	"github.com/gin-gonic/gin"
	"github.com/mcorbin/fizz"
	"github.com/mcorbin/fizz/openapi"
	"github.com/mcorbin/gadgeto/tonic"
	prom "github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

type Component struct {
	Config *Configuration
	Logger *zap.Logger
	Router *gin.Engine
	Fizz   *fizz.Fizz
	Server *http.Server

	requestHistogram *prom.HistogramVec
	responseCounter  *prom.CounterVec
	wg               sync.WaitGroup
}

func New(logger *zap.Logger, config *Configuration) (*Component, error) {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	tonic.SetErrorHook(ErrorHook(logger))
	tonic.SetBindHook(DefaultBindingHookMaxBodyBytes(tonic.DefaultMaxBodyBytes))
	router.Use(ginzap.RecoveryWithZap(logger, true))
	router.NoRoute(func(c *gin.Context) {
		c.JSON(404, NewResponse("Not found"))
	})
	f := fizz.NewFromEngine(router)
	// authorized := router.Group("/", gin.BasicAuth(gin.Accounts{
	// 	config.BasicAuth.Username: config.BasicAuth.Password,
	// }))

	address := fmt.Sprintf("%s:%d", config.Host, config.Port)
	server := &http.Server{
		Addr:    address,
		Handler: f,
	}
	respCounter := prom.NewCounterVec(
		prom.CounterOpts{
			Name: "http_responses_total",
			Help: "Count the number of HTTP responses.",
		},
		[]string{"method", "status", "path"})

	buckets := []float64{
		0.05, 0.1, 0.2, 0.4, 0.8, 1,
		1.5, 2, 3, 5}

	reqHistogram := prom.NewHistogramVec(
		prom.HistogramOpts{
			Name:    "http_requests_duration_second",
			Help:    "Time to execute http requests",
			Buckets: buckets,
		},
		[]string{"method", "path"})

	component := Component{
		Fizz:             f,
		Config:           config,
		Router:           router,
		Server:           server,
		Logger:           logger,
		requestHistogram: reqHistogram,
		responseCounter:  respCounter,
	}
	return &component, nil
}

// Start starts the http server
func (c *Component) Start() error {
	infos := &openapi.Info{
		Title:       "AppClacks API",
		Description: `This is the AppClacks API.`,
		Version:     "0.1.0",
	}
	c.Logger.Info(fmt.Sprintf("Starting the HTTP server component on %s:%d", c.Config.Host, c.Config.Port))
	c.Fizz.Generator().UseFullSchemaNames(false)

	c.Fizz.GET("/healthz", nil, c.health)
	c.Fizz.GET("/openapi.json", nil, c.Fizz.OpenAPI(infos, "json"))
	c.Fizz.GET("/openapi.yaml", nil, c.Fizz.OpenAPI(infos, "yaml"))
	tonic.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "-" {
			return ""
		}
		return name
	})
	go func() {
		defer c.wg.Done()
		if err := c.Server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			c.Logger.Error(fmt.Sprintf("HTTP server error: %s", err.Error()))
			os.Exit(2)
		}
	}()
	c.wg.Add(1)
	return nil
}

// Stop stop the server compoment
func (c *Component) Stop() error {
	c.Logger.Info("Stopping the HTTP server component")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err := c.Server.Shutdown(ctx)
	c.wg.Wait()
	if err != nil {
		c.Logger.Error(err.Error())
		return err
	}
	c.Logger.Info("HTTP server stopped")
	return nil
}
