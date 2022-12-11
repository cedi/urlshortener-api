package router

import (
	"fmt"
	"net/http"

	docs "github.com/cedi/urlshortener-api/docs"
	urlshortenerController "github.com/cedi/urlshortener-api/pkg/controller"

	"github.com/gin-gonic/contrib/secure"
	"github.com/gin-gonic/gin"

	"github.com/go-logr/logr"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"

	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func NewGinGonicHTTPServer(setupLog *logr.Logger, bindAddr string) (*gin.Engine, *http.Server) {
	router := gin.New()
	router.Use(
		otelgin.Middleware("urlshortener-api"),
		secure.Secure(secure.Options{
			SSLRedirect:           true,
			SSLProxyHeaders:       map[string]string{"X-Forwarded-Proto": "https"},
			STSIncludeSubdomains:  true,
			FrameDeny:             true,
			ContentTypeNosniff:    true,
			BrowserXssFilter:      true,
			ContentSecurityPolicy: "default-src 'self' data: 'unsafe-inline'",
		}),
	)

	//load html file
	router.LoadHTMLGlob("html/templates/*.html")

	//static path
	router.Static("assets", "./html/assets")

	setupLog.Info(fmt.Sprintf("Starting gin-tonic router on binAddr: '%s'", bindAddr))
	srv := &http.Server{
		Addr:    bindAddr,
		Handler: router,
	}

	docs.SwaggerInfo.BasePath = "/"

	return router, srv
}

func Load(router *gin.Engine, shortlinkController *urlshortenerController.ShortlinkController) {
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	v1 := router.Group("/api/v1")
	loadV1Routes(v1, shortlinkController)
}

func loadV1Routes(v1 *gin.RouterGroup, shortlinkController *urlshortenerController.ShortlinkController) {
	v1.GET("/shortlink/", shortlinkController.HandleListShortLink)
	v1.GET("/shortlink/:shortlink", shortlinkController.HandleGetShortLink)
	v1.POST("/shortlink/:shortlink", shortlinkController.HandleCreateShortLink)
	v1.PUT("/shortlink/:shortlink", shortlinkController.HandleUpdateShortLink)
	v1.DELETE("/shortlink/:shortlink", shortlinkController.HandleDeleteShortLink)
}
