package controller

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gabriel-vasile/mimetype"
	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
	"github.com/michibiki-io/simple-http-fileserver/server/model"
	"github.com/michibiki-io/simple-http-fileserver/server/service"
	"github.com/michibiki-io/simple-http-fileserver/server/utility"
)

var thumbnailWidth int = 256

var thumbnailWidthRetina int = 512

func init() {

	thumbnailWidth = utility.GetIntEnv("THUMBNAIL_WIDTH", thumbnailWidth)
	thumbnailWidthRetina = utility.GetIntEnv("THUMBNAIL_WIDTH_RETINA", thumbnailWidthRetina)

}

func CreateTransparencyFileSystemHandler(urlPrefix, transparentRule string, fs static.ServeFileSystem) gin.HandlerFunc {
	fileserver := http.FileServer(fs)
	if urlPrefix != "" {
		fileserver = http.StripPrefix(urlPrefix, fileserver)
	}
	return func(c *gin.Context) {
		url := strings.TrimPrefix(c.Request.URL.Path, urlPrefix)
		if url == transparentRule {
			c.Next()
		} else if fs.Exists(urlPrefix, c.Request.URL.Path) {
			fileserver.ServeHTTP(c.Writer, c.Request)
			c.Abort()
		}
	}
}

func FileSystemHandler(contextPath, prefixFilePath string, directoryFileHandler func(*gin.Context, http.File, string, string)) gin.HandlerFunc {

	fileSystem := service.DotFileHidingFileSystem(http.Dir(prefixFilePath))
	fileServer := http.StripPrefix(contextPath+prefixFilePath, http.FileServer(fileSystem))

	return func(c *gin.Context) {

		filePath := c.Param("filepath")
		node, err := fileSystem.Open(filePath)
		if err != nil {
			if c.Request.Method == "GET" {
				c.HTML(http.StatusOK, "files", gin.H{"absoluteFilePath": filePath, "contextPath": utility.GetContextPath()})
				c.Abort()
			} else {
				c.JSON(http.StatusNotFound, gin.H{"error": "file not found"})
				c.Abort()
			}
		} else {
			defer node.Close()
			if fileInfo, err := node.Stat(); err != nil {
				if c.Request.Method == "GET" {
					c.HTML(http.StatusOK, "files", gin.H{"absoluteFilePath": filePath, "contextPath": utility.GetContextPath()})
					c.Abort()
				} else {
					c.JSON(http.StatusNotFound, gin.H{"error": "file not found"})
					c.Abort()
				}
			} else if fileInfo.IsDir() {
				directoryFileHandler(c, node, prefixFilePath, filePath)
			} else {
				buff := make([]byte, 512)
				var mimeType *mimetype.MIME = nil
				if count, err := node.Read(buff); err == nil && count > 0 {
					if mimeType = mimetype.Detect(buff); err == nil {
						if strings.HasPrefix(mimeType.String(), "image/") {
							if strings.HasPrefix(c.Query("type"), "thumb") {
								thumbType := c.Query("type")
								imageType := strings.TrimSuffix(mimeType.String(), "image/")
								allData := make([]byte, 0, fileInfo.Size())
								if data, err := io.ReadAll(node); err == nil {
									allData = append(allData, buff...)
									allData = append(allData, data...)
									if imageType != "svg+xml" {
										width, height := getThumbnailImageSize(thumbType)
										if thumb, err := utility.CreateThumbnailImage(width, height, allData); err == nil {
											c.Data(http.StatusOK, mimeType.String(), thumb)
											c.Abort()
											return
										}
									}
									c.Data(http.StatusOK, mimeType.String(), allData)
									c.Abort()
									return
								}
							}
						} else if strings.HasPrefix(mimeType.String(), "text/html") {
							if strings.HasPrefix(c.Query("type"), "thumb") {
								c.Data(http.StatusOK, "image/svg+xml", readSvgIconFile(mimeType))
								c.Abort()
								return
							} else {
								charaSet := strings.TrimPrefix(mimeType.String(), "text/html")
								allData := make([]byte, 0, fileInfo.Size())
								if data, err := io.ReadAll(node); err == nil {
									allData = append(allData, buff...)
									allData = append(allData, data...)
									c.Data(http.StatusOK, "text/plain"+charaSet, allData)
									c.Abort()
									return
								}
							}
						}
					}
				}
				if strings.HasPrefix(c.Query("type"), "thumb") {
					c.Data(http.StatusOK, "image/svg+xml", readSvgIconFile(mimeType))
					c.Abort()
					return
				} else {
					fileServer.ServeHTTP(c.Writer, c.Request)
				}
			}
		}
	}
}

func IsFolderPublicHandler(contextPath, prefix string) gin.HandlerFunc {

	return func(c *gin.Context) {
		url := strings.TrimPrefix(c.Request.URL.Path, contextPath+prefix)

		// check permission
		authorized := []string{}
		if authorized = isPermitted(url); len(authorized) == 0 {
			if defaultPermission {
				c.Set("isPublic", true)
			}
		} else if len(authorized) == 1 && authorized[0] == "**" {
			c.Set("isPublic", true)
		}
	}
}

func FolderPermissionHandler(contextPath, prefix string) gin.HandlerFunc {

	return func(c *gin.Context) {

		if tmp, ok := c.Get("isPublic"); ok {
			if isPublic, _ := tmp.(bool); isPublic {
				c.Next()
				return
			}
		}

		url := strings.TrimPrefix(c.Request.URL.Path, contextPath+prefix)

		// check permission
		authorized := []string{}
		if authorized = isPermitted(url); len(authorized) == 0 {
			if defaultPermission {
				c.Next()
			} else {
				c.Set("authorization", model.AuthorizationState{StatusCode: http.StatusForbidden})
				c.Next()
			}
		} else if len(authorized) == 1 && authorized[0] == "*" {
			c.Next()
		} else {
			// get user model from previous handler
			userModel := model.User{}
			if user, ok := c.Get("user"); ok {
				userModel, _ = user.(model.User)
			}

			// check authorized
			for _, v := range authorized {
				if userModel.Id == v {
					c.Next()
					return
				} else if utility.StringsContains(userModel.Groups, v) {
					c.Next()
					return
				}
			}

			if c.Request.Method == "GET" {
				c.HTML(http.StatusOK, "files", gin.H{"absoluteFilePath": prefix + url, "contextPath": utility.GetContextPath()})
				c.Abort()
			} else {
				c.Set("authorization", model.AuthorizationState{StatusCode: http.StatusForbidden})
				c.Next()
			}
		}
	}
}

type fileInfo struct {
	Path  string `json:"path"`
	IsDir bool   `json:"isDir"`
	Size  int64  `json:"size"`
}

func GetDirectoryFileHandler(c *gin.Context, node http.File, prefixFilePath string, filePath string) {

	if !strings.HasSuffix(filePath, "/") {
		c.Redirect(http.StatusTemporaryRedirect, c.Request.URL.Path+"/")
	} else {
		c.HTML(http.StatusOK, "files", gin.H{"absoluteFilePath": prefixFilePath + filePath, "contextPath": utility.GetContextPath()})
	}

}

func PostDirectoryFileHandler(c *gin.Context, node http.File, prefixFilePath string, filePath string) {
	if auth, ok := c.Get("authorization"); ok {
		if auth, ok = auth.(model.AuthorizationState); ok {
			c.JSON(auth.(model.AuthorizationState).StatusCode, gin.H{"error": "there are no private folders"})
			return
		}
	}
	if fileInfoList, err := node.Readdir(-1); err != nil {
		if c.Request.Method == "GET" {
			c.HTML(http.StatusOK, "files", gin.H{"absoluteFilePath": "", "contextPath": utility.GetContextPath()})
			c.Abort()
		} else {
			c.Set("authorization", model.AuthorizationState{StatusCode: http.StatusNotFound})
			c.Next()
		}
	} else {
		fileInList := []fileInfo{}
		for _, ifo := range fileInfoList {
			size := int64(0)
			if !ifo.IsDir() {
				size = ifo.Size()
			}
			fileInfo := fileInfo{Path: ifo.Name(), IsDir: ifo.IsDir(), Size: size}
			fileInList = append(fileInList, fileInfo)
		}
		breadcrumbList := strings.Split(strings.TrimSuffix(strings.TrimPrefix(prefixFilePath+filePath, string(filepath.Separator)), string(filepath.Separator)), string(filepath.Separator))
		breadcrumbLen := len(breadcrumbList)
		breadcrumb := make([][2]string, breadcrumbLen, breadcrumbLen)
		dotList := make([]byte, 0, 20)
		for i := 0; i < breadcrumbLen; i++ {
			breadcrumb[breadcrumbLen-1-i] = [2]string{breadcrumbList[breadcrumbLen-1-i], string(dotList)}
			dotList = append(dotList, "../"...)
		}
		c.JSON(http.StatusOK, gin.H{"list": fileInList, "breadcrumb": breadcrumb})
	}
}

func getThumbnailImageSize(typeName string) (width, height int) {

	width = 128
	height = 96

	if typeName == "thumbx2" {
		width *= 2
		height *= 2
	} else if typeName == "thumbx4" {
		width *= 4
		height *= 4
	}

	return
}

func readSvgIconFile(mimetype *mimetype.MIME) (image []byte) {

	image = make([]byte, 0)
	mimetypeString := ""

	if mimetype != nil {
		mimetypeString = mimetype.String()
	}

	switch mimetypeString {
	case "application/pdf":
		image, _ = os.ReadFile("./static/img/file-pdf-regular.svg")
	case "text/css", "text/html", "text/javascript", "application/xml", "text/xml":
		image, _ = os.ReadFile("./static/img/file-code-regular.svg")
	// case "application/msword":
	// 	image, _ = os.ReadFile("./static/img/file-word-regular.svg")
	// case "application/vnd.openxmlformats-officedocument.wordprocessingml.document":
	// 	image, _ = os.ReadFile("./static/img/file-word-regular.svg")
	// case "application/vnd.ms-excel":
	// 	image, _ = os.ReadFile("./static/img/file-excel-regular.svg")
	// case "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet":
	// 	image, _ = os.ReadFile("./static/img/file-excel-regular.svg")
	// case "application/vnd.ms-powerpoint":
	// 	image, _ = os.ReadFile("./static/img/file-powerpoint-regular.svg")
	// case "application/vnd.openxmlformats-officedocument.presentationml.presentation":
	// 	image, _ = os.ReadFile("./static/img/file-powerpoint-regular.svg")
	default:
		if strings.HasPrefix(mimetypeString, "text") {
			image, _ = os.ReadFile("./static/img/file-alt-regular.svg")
		} else {
			image, _ = os.ReadFile("./static/img/file-regular.svg")
		}
	}

	return
}
