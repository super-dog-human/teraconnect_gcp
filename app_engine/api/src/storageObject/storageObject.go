package storageObject

import (
	"cloudHelper"
	"encoding/json"
	"lessonType"
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo"
	"github.com/pkg/errors"
	"github.com/rs/xid"
	"google.golang.org/appengine"
	"google.golang.org/appengine/datastore"
	"google.golang.org/appengine/log"
)

const bucketName = "teraconn_material"

// Gets is get signed URLs of files.
func Gets(c echo.Context) error {
	ctx := appengine.NewContext(c.Request())

	jsonString := c.Request().Header.Get("X-Get-Params")
	var fileRequests []fileRequest
	if err := json.Unmarshal([]byte(jsonString), &fileRequests); err != nil {
		log.Errorf(ctx, "%+v\n", errors.WithStack(err))
		return c.JSON(http.StatusInternalServerError, err.Error())
	}

	urlLength := len(fileRequests)
	urls := make([]signedURL, urlLength)

	for i, request := range fileRequests {
		// TODO check user permission
		// TODO check file exists

		filePath := filePath(request.Entity, request.ID, request.Extension)
		url, err := cloudHelper.GetGCSSignedURL(ctx, bucketName, filePath, "GET", "")
		if err != nil {
			log.Errorf(ctx, "%+v\n", errors.WithStack(err))
			return c.JSON(http.StatusInternalServerError, err.Error())
		}
		urls[i] = signedURL{request.ID, url}
	}

	return c.JSON(http.StatusOK, urlResponses{SignedURLs: urls})
}

// Posts is create blank object to Cloud Storage for direct upload from client.
func Posts(c echo.Context) error {
	ctx := appengine.NewContext(c.Request())

	request := new(postRequest)
	if err := c.Bind(request); err != nil {
		log.Errorf(ctx, "%+v\n", errors.WithStack(err))
		return c.JSON(http.StatusInternalServerError, err.Error())
	}

	urls := make([]signedURL, len(request.FileRequests))

	for i, fileRequest := range request.FileRequests {
		fileID := xid.New().String()
		filePath := filePath(fileRequest.Entity, fileID, fileRequest.Extension)

		if err := cloudHelper.CreateObjectToGCS(ctx, bucketName, filePath, fileRequest.ContentType, nil); err != nil {
			log.Errorf(ctx, "%+v\n", errors.WithStack(err))
			return c.JSON(http.StatusInternalServerError, err.Error())
		}

		url, err := cloudHelper.GetGCSSignedURL(ctx, bucketName, filePath, "PUT", fileRequest.ContentType)
		if err != nil {
			log.Errorf(ctx, "%+v\n", errors.WithStack(err))
			return c.JSON(http.StatusInternalServerError, err.Error())
		}

		if fileRequest.Entity == "graphic" {
			graphic := new(lessonType.Graphic)
			graphic.Created = time.Now()
			graphic.FileType = fileRequest.Extension
			// graphic.UserID  = "foo"	// TODO
			key := datastore.NewKey(ctx, "Graphic", fileID, 0, nil)
			if _, err = datastore.Put(ctx, key, graphic); err != nil {
				log.Errorf(ctx, "%+v\n", errors.WithStack(err))
				return c.JSON(http.StatusInternalServerError, err.Error())
			}
		} else if fileRequest.Entity == "avatar" {
			avatar := new(lessonType.Avatar)
			avatar.Created = time.Now()
			// avatar.UserID  = "foo"	// TODO
			key := datastore.NewKey(ctx, "Avatar", fileID, 0, nil)
			if _, err = datastore.Put(ctx, key, avatar); err != nil {
				log.Errorf(ctx, "%+v\n", errors.WithStack(err))
				return c.JSON(http.StatusInternalServerError, err.Error())
			}
		}

		urls[i] = signedURL{fileID, url}
	}

	return c.JSON(http.StatusOK, urlResponses{SignedURLs: urls})
}

func filePath(entity string, id string, extension string) string {
	return strings.ToLower(entity) + "/" + id + "." + extension
}

type postRequest struct {
	FileRequests []fileRequest `json:"fileRequests"`
}

type fileRequest struct {
	ID          string `json:"id"`
	Entity      string `json:"entity"`
	Extension   string `json:"extension"`
	ContentType string `json:"contentType"`
}

type urlResponses struct {
	SignedURLs []signedURL `json:"signedURLs"`
}

type signedURL struct {
	FileID    string `json:"fileID"`
	SignedURL string `json:"signedURL"`
}
