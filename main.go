package main

import (
	"net/http"

	"github.com/gin-contrib/multitemplate"
	"github.com/gin-gonic/gin"
	"github.com/michibiki-io/simple-http-fileserver/server/controller"
	"github.com/michibiki-io/simple-http-fileserver/server/service"
)

func main() {

	engine := gin.Default()
	engine.LoadHTMLGlob("templates/*.tmpl")
	engine.HTMLRender = createRender()
	engine.Use(controller.CreateTransparencyFileSystemHandler("/", "", service.DotFileHidingFileSystem(http.Dir("static"))))
	if handler, err := controller.CreateSessionHandler(); err != nil {
		panic(err)
	} else {
		engine.Use(handler)
	}

	root := engine.Group("/")
	{
		root.GET("/", func(c *gin.Context) {
			c.HTML(http.StatusOK, "index", gin.H{})
		})

		public := root.Group("/public")
		{
			public.GET("/*filepath", controller.FileSystemHandler("/public", controller.GetDirectoryFileHandler))
			public.HEAD("/*filepath", controller.FileSystemHandler("/public", controller.GetDirectoryFileHandler))
			public.POST("/*filepath", controller.FileSystemHandler("/public", controller.PostDirectoryFileHandler))
		}

		private := root.Group("/private").Use(
			controller.FromSessionToStoreHandler("user", "user"),
			controller.FromSessionToStoreHandler("tokens", "tokens"),
			controller.ProcessAccessToken,
			controller.VerifyAuth,
			controller.FromStoreToSessionHandler("user", "user"),
			controller.FromStoreToSessionHandler("tokens", "tokens"),
			controller.ShowAuthorizeInterfaceHander("/v1/login/ui"),
			controller.FolderPermissionHandler("/private"))
		{
			private.GET("/*filepath", controller.FileSystemHandler("/private", controller.GetDirectoryFileHandler))
			private.HEAD("/*filepath", controller.FileSystemHandler("/private", controller.GetDirectoryFileHandler))
			private.POST("/*filepath", controller.FileSystemHandler("/private", controller.PostDirectoryFileHandler))
		}

		root.GET("/token",
			controller.FromSessionToStoreHandler("apitokens", "tokens"),
			controller.VerifyAuth,
			controller.FromStoreToSessionHandler("tokens", "apitokens"),
			controller.ShowAuthorizeInterfaceHander("/v1/login/token"),
			func(c *gin.Context) {
				c.HTML(http.StatusOK, "token", gin.H{})
			})
	}

	v1 := engine.Group("/v1")
	{
		login := v1.Group("/login")
		{
			login.POST("/ui",
				controller.Authorize,
				controller.FromStoreToSessionHandler("tokens", "tokens"),
				controller.JsonResponceHandler("response"))
			login.POST("/token",
				controller.Authorize,
				controller.FromStoreToSessionHandler("tokens", "apitokens"),
				controller.JsonResponceHandler("response"))
			login.POST("/api",
				controller.AuthorizeToken,
				controller.FromStoreToSessionHandler("tokens", "tokens"),
				controller.JsonResponceHandler("response"))
			login.POST("/status",
				controller.FromSessionToStoreHandler("user", "user"),
				controller.StatusCheck)
		}

		v1.POST("/token",
			controller.FromSessionToStoreHandler("apitokens", "tokens"),
			controller.RequestApiToken,
			controller.FromStoreToSessionHandler("tokens", "apitokens"),
			controller.JsonResponceHandler("response"))

		refresh := v1.Group("/refresh")
		{
			refresh.POST("/ui",
				controller.FromSessionToStoreHandler("tokens", "tokens"),
				controller.ProcessRefreshToken,
				controller.Refresh,
				controller.FromStoreToSessionHandler("user", "user"),
				controller.FromStoreToSessionHandler("tokens", "tokens"))
			refresh.POST("/api",
				controller.FromSessionToStoreHandler("apitokens", "tokens"),
				controller.ProcessRefreshToken,
				controller.Refresh,
				controller.FromStoreToSessionHandler("tokens", "apitokens"),
				controller.JsonResponceHandler("response"))
		}

		v1.POST("/logout",
			controller.FromSessionToStoreHandler("user", "user"),
			controller.FromSessionToStoreHandler("tokens", "tokens"),
			controller.ProcessAccessToken,
			controller.Deauthorize,
			controller.ClearSession)
	}
	engine.Run(":80")
}

func createRender() multitemplate.Renderer {
	r := multitemplate.NewRenderer()
	r.AddFromFiles("index", "templates/layout.tmpl", "templates/index.tmpl")
	r.AddFromFiles("files", "templates/layout.tmpl", "templates/filelist.tmpl")
	r.AddFromFiles("login", "templates/layout.tmpl", "templates/login.tmpl")
	r.AddFromFiles("token", "templates/layout.tmpl", "templates/token.tmpl")
	return r
}
