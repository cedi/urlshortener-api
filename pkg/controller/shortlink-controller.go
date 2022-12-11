package controller

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	shortlinkClient "github.com/cedi/urlshortener-api/pkg/client"
	"github.com/cedi/urlshortener-api/pkg/observability"
	"github.com/cedi/urlshortener/api/v1alpha1"
	"github.com/go-logr/logr"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// ShortlinkController is an object who handles the requests made towards our shortlink-application
type ShortlinkController struct {
	log    *logr.Logger
	tracer trace.Tracer
	client *shortlinkClient.ShortlinkClient
}

// NewShortlinkController creates a new ShortlinkController
func NewShortlinkController(log *logr.Logger, tracer trace.Tracer, client *shortlinkClient.ShortlinkClient) *ShortlinkController {
	return &ShortlinkController{
		log:    log,
		tracer: tracer,
		client: client,
	}
}

// HandleListShortLink handles the listing of
// @BasePath /api/v1/
// @Summary       list shortlinks
// @Schemes       http https
// @Description   list shortlinks
// @Produce       text/plain
// @Produce       application/json
// @Success       200         {object} []ShortLink "Success"
// @Failure       404         {object} int         "NotFound"
// @Failure       500         {object} int         "InternalServerError"
// @Tags api/v1/
// @Router /api/v1/shortlink/ [get]
func (s *ShortlinkController) HandleListShortLink(c *gin.Context) {
	contentType := c.Request.Header.Get("accept")

	// Call the HTML method of the Context to render a template
	ctx, span := s.tracer.Start(c.Request.Context(), "ShortlinkController.HandleListShortLink", trace.WithAttributes(attribute.String("accepted_content_type", contentType)))
	defer span.End()

	shortlinkList, err := s.client.List(ctx)
	if err != nil {
		observability.RecordError(span, s.log, err, "Failed to list ShortLink")

		statusCode := http.StatusInternalServerError

		if strings.Contains(err.Error(), "not found") {
			statusCode = http.StatusNotFound
		}

		ginReturnError(c, statusCode, contentType, err.Error())
		return
	}

	targetList := make([]ShortLink, len(shortlinkList.Items))

	for idx, shortlink := range shortlinkList.Items {
		targetList[idx] = ShortLink{
			Name:   shortlink.ObjectMeta.Name,
			Spec:   shortlink.Spec,
			Status: shortlink.Status,
		}
	}

	if contentType == ContentTypeApplicationJSON {
		c.JSON(http.StatusOK, targetList)
	} else if contentType == ContentTypeTextPlain {
		shortLinks := ""
		for _, shortlink := range targetList {
			shortLinks += fmt.Sprintf("%s: %s\n", shortlink.Name, shortlink.Spec.Target)
		}
		c.Data(http.StatusOK, contentType, []byte(shortLinks))
	}
}

// HandleGetShortLink returns the shortlink
// @BasePath      /api/v1/
// @Summary       get a shortlink
// @Schemes       http https
// @Description   get a shortlink
// @Produce       text/plain
// @Produce       application/json
// @Param         shortlink   path      string  false          "the shortlink URL part (shortlink id)" example(home)
// @Success       200         {object}  ShortLink "Success"
// @Failure       404         {object}  int       "NotFound"
// @Failure       500         {object}  int       "InternalServerError"
// @Tags api/v1/
// @Router /api/v1/shortlink/{shortlink} [get]
func (s *ShortlinkController) HandleGetShortLink(c *gin.Context) {
	shortlinkName := c.Param("shortlink")

	contentType := c.Request.Header.Get("accept")

	// Call the HTML method of the Context to render a template
	ctx, span := s.tracer.Start(c.Request.Context(), "ShortlinkController.HandleGetShortLink", trace.WithAttributes(attribute.String("shortlink", shortlinkName), attribute.String("accepted_content_type", contentType)))
	defer span.End()

	shortlink, err := s.client.Get(ctx, shortlinkName)
	if err != nil {
		observability.RecordError(span, s.log, err, "Failed to get ShortLink")

		statusCode := http.StatusInternalServerError

		if strings.Contains(err.Error(), "not found") {
			statusCode = http.StatusNotFound
		}

		ginReturnError(c, statusCode, contentType, err.Error())
		return
	}

	if contentType == ContentTypeTextPlain {
		c.Data(http.StatusOK, contentType, []byte(shortlink.Spec.Target))
	} else if contentType == ContentTypeApplicationJSON {
		c.JSON(http.StatusOK, ShortLink{
			Name:   shortlink.Name,
			Spec:   shortlink.Spec,
			Status: shortlink.Status,
		})
	}
}

// HandleCreateShortLink handles the creation of a shortlink and redirects according to the configuration
// @BasePath /api/v1/
// @Summary       create new shortlink
// @Schemes       http https
// @Description   create a new shortlink
// @Accept        application/json
// @Produce       text/plain
// @Produce       application/json
// @Param         shortlink   path      string                 	false  					"the shortlink URL part (shortlink id)" example(home)
// @Param         spec        body      v1alpha1.ShortLinkSpec 	true   					"shortlink spec"
// @Success       200         {object}  int     				"Success"
// @Success       301         {object}  int     				"MovedPermanently"
// @Success       302         {object}  int     				"Found"
// @Success       307         {object}  int     				"TemporaryRedirect"
// @Success       308         {object}  int     				"PermanentRedirect"
// @Failure       404         {object}  int     				"NotFound"
// @Failure       500         {object}  int     				"InternalServerError"
// @Tags api/v1/
// @Router /api/v1/shortlink/{shortlink} [post]
func (s *ShortlinkController) HandleCreateShortLink(c *gin.Context) {
	shortlinkName := c.Param("shortlink")
	contentType := c.Request.Header.Get("accept")

	// Call the HTML method of the Context to render a template
	ctx, span := s.tracer.Start(c.Request.Context(), "ShortlinkController.HandleGetShortLink", trace.WithAttributes(attribute.String("shortlink", shortlinkName), attribute.String("accepted_content_type", contentType)))
	defer span.End()

	shortlink := v1alpha1.ShortLink{
		ObjectMeta: v1.ObjectMeta{
			Name: shortlinkName,
		},
		Spec: v1alpha1.ShortLinkSpec{},
	}

	jsonData, err := ioutil.ReadAll(c.Request.Body)
	if err != nil {
		observability.RecordError(span, s.log, err, "Failed to read request-body")
		ginReturnError(c, http.StatusInternalServerError, contentType, err.Error())
		return
	}

	if err := json.Unmarshal([]byte(jsonData), &shortlink.Spec); err != nil {
		observability.RecordError(span, s.log, err, "Failed to read spec-json")
		ginReturnError(c, http.StatusInternalServerError, contentType, err.Error())
		return
	}

	if err := s.client.Create(ctx, &shortlink); err != nil {
		observability.RecordError(span, s.log, err, "Failed to create ShortLink")
		ginReturnError(c, http.StatusInternalServerError, contentType, err.Error())
		return
	}

	if contentType == ContentTypeTextPlain {
		c.Data(http.StatusOK, contentType, []byte(fmt.Sprintf("%s: %s\n", shortlink.Name, shortlink.Spec.Target)))
	} else if contentType == ContentTypeApplicationJSON {
		c.JSON(http.StatusOK, ShortLink{
			Name:   shortlink.Name,
			Spec:   shortlink.Spec,
			Status: shortlink.Status,
		})
	}
}

// HandleDeleteShortLink handles the update of a shortlink
// @BasePath /api/v1/
// @Summary       update existing shortlink
// @Schemes       http https
// @Description   update a new shortlink
// @Accept        application/json
// @Produce       text/plain
// @Produce       application/json
// @Param         shortlink   path      string                 true   "the shortlink URL part (shortlink id)" example(home)
// @Param         spec        body      v1alpha1.ShortLinkSpec true   "shortlink spec"
// @Success       200         {object}  int     "Success"
// @Failure       404         {object}  int     "NotFound"
// @Failure       500         {object}  int     "InternalServerError"
// @Tags api/v1/
// @Router /api/v1/shortlink/{shortlink} [put]
func (s *ShortlinkController) HandleUpdateShortLink(c *gin.Context) {
	shortlinkName := c.Param("shortlink")

	contentType := c.Request.Header.Get("accept")

	// Call the HTML method of the Context to render a template
	ctx, span := s.tracer.Start(c.Request.Context(), "ShortlinkController.HandleGetShortLink", trace.WithAttributes(attribute.String("shortlink", shortlinkName), attribute.String("accepted_content_type", contentType)))
	defer span.End()

	shortlink, err := s.client.Get(ctx, shortlinkName)
	if err != nil {
		observability.RecordError(span, s.log, err, "Failed to get ShortLink")

		statusCode := http.StatusInternalServerError

		if strings.Contains(err.Error(), "not found") {
			statusCode = http.StatusNotFound
		}

		ginReturnError(c, statusCode, contentType, err.Error())
		return
	}

	shortlinkSpec := v1alpha1.ShortLinkSpec{}

	jsonData, err := ioutil.ReadAll(c.Request.Body)
	if err != nil {
		observability.RecordError(span, s.log, err, "Failed to read request-body")

		ginReturnError(c, http.StatusInternalServerError, contentType, err.Error())
		return
	}

	if err := json.Unmarshal([]byte(jsonData), &shortlinkSpec); err != nil {
		observability.RecordError(span, s.log, err, "Failed to read ShortLink Spec JSON")

		ginReturnError(c, http.StatusInternalServerError, contentType, err.Error())
		return
	}

	shortlink.Spec = shortlinkSpec

	if err := s.client.Update(ctx, shortlink); err != nil {
		observability.RecordError(span, s.log, err, "Failed to update ShortLink")

		ginReturnError(c, http.StatusInternalServerError, contentType, err.Error())
		return
	}

	ginReturnError(c, http.StatusOK, contentType, "")
}

// HandleDeleteShortLink handles the deletion of a shortlink
// @BasePath /api/v1/
// @Summary       delete shortlink
// @Schemes       http https
// @Description   delete shortlink
// @Produce       text/plain
// @Produce       application/json
// @Param         shortlink   path      string                 true   "the shortlink URL part (shortlink id)" example(home)
// @Success       200         {object}  int     "Success"
// @Failure       404         {object}  int     "NotFound"
// @Failure       500         {object}  int     "InternalServerError"
// @Tags api/v1/
// @Router /api/v1/shortlink/{shortlink} [delete]
func (s *ShortlinkController) HandleDeleteShortLink(c *gin.Context) {
	shortlinkName := c.Param("shortlink")

	contentType := c.Request.Header.Get("accept")

	// Call the HTML method of the Context to render a template
	ctx, span := s.tracer.Start(c.Request.Context(), "ShortlinkController.HandleGetShortLink", trace.WithAttributes(attribute.String("shortlink", shortlinkName), attribute.String("accepted_content_type", contentType)))
	defer span.End()

	shortlink, err := s.client.Get(ctx, shortlinkName)
	if err != nil {
		observability.RecordError(span, s.log, err, "Failed to get ShortLink")

		statusCode := http.StatusInternalServerError

		if strings.Contains(err.Error(), "not found") {
			statusCode = http.StatusNotFound
		}

		ginReturnError(c, statusCode, contentType, err.Error())
		return
	}

	if err := s.client.Delete(ctx, shortlink); err != nil {
		statusCode := http.StatusInternalServerError

		if strings.Contains(err.Error(), "not found") {
			statusCode = http.StatusNotFound
		}

		observability.RecordError(span, s.log, err, "Failed to delete ShortLink")

		ginReturnError(c, statusCode, contentType, err.Error())
		return
	}
}
