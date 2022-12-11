package controller

import (
	"github.com/cedi/urlshortener/api/v1alpha1"
	"github.com/gin-gonic/gin"
)

const (
	ContentTypeApplicationJSON = "application/json"
	ContentTypeTextPlain       = "text/plain"
)

type ShortLink struct {
	Name   string                   `json:"name"`
	Spec   v1alpha1.ShortLinkSpec   `json:"spec,omitempty"`
	Status v1alpha1.ShortLinkStatus `json:"status,omitempty"`
}

type JsonReturnError struct {
	Code  int    `json:"code"`
	Error string `json:"error"`
}

func ginReturnError(c *gin.Context, statusCode int, contentType string, err string) {
	if contentType == ContentTypeTextPlain {
		c.Data(statusCode, contentType, []byte(err))
	} else if contentType == ContentTypeApplicationJSON {
		c.JSON(statusCode, JsonReturnError{
			Code:  statusCode,
			Error: err,
		})
	}
}
