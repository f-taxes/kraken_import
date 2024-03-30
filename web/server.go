package web

import (
	"context"
	"embed"
	"net/http"
	"time"

	"github.com/f-taxes/kraken_import/conf"
	"github.com/f-taxes/kraken_import/fetcher"
	g "github.com/f-taxes/kraken_import/global"
	"github.com/f-taxes/kraken_import/grpc_client"
	iu "github.com/f-taxes/kraken_import/irisutils"
	"github.com/kataras/golog"
	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/view"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func Start(address string, webAssets embed.FS) {
	if conf.App.Bool("debug") {
		g.SetGoLogDebugFormat()
		golog.SetLevel("debug")
		golog.Info("Debug logging is enabled!")
	}

	app := iris.New()
	app.Use(iris.Compression)
	app.SetRoutesNoLog(true)

	registerFrontend(app, webAssets)

	app.Get("/settings", func(ctx iris.Context) {
		settings, err := grpc_client.GrpcClient.GetSettings(context.Background())
		if err != nil {
			golog.Error(err)
			ctx.JSON(iu.Resp{
				Result: false,
			})
			return
		}

		ctx.JSON(iu.Resp{
			Result: true,
			Data:   settings,
		})
	})

	app.Get("/account/list", func(ctx iris.Context) {
		accounts := []g.Account{}
		conf.App.BindStruct("accounts", &accounts)

		// accounts[0].LastFetched = time.Now().UTC().Format(time.RFC3339Nano)
		// conf.App.Set("accounts", accounts)
		// conf.WriteAppConfig()

		ctx.JSON(iu.Resp{
			Result: true,
			Data:   accounts,
		})
	})

	app.Post("/account/add", func(ctx iris.Context) {
		reqData := g.Account{}

		if !iu.ReadJSON(ctx, &reqData) {
			return
		}

		if reqData.ID == "" {
			reqData.ID = primitive.NewObjectID().Hex()
		}

		accounts := []g.Account{}
		conf.App.BindStruct("accounts", &accounts)
		accounts = append(accounts, reqData)
		conf.App.Set("accounts", accounts)
		conf.WriteAppConfig()

		ctx.JSON(iu.Resp{
			Result: true,
		})
	})

	app.Post("/account/add", func(ctx iris.Context) {
		reqData := g.Account{}

		if !iu.ReadJSON(ctx, &reqData) {
			return
		}

		if reqData.ID == "" {
			reqData.ID = primitive.NewObjectID().Hex()
		}

		accounts := []g.Account{}
		conf.App.BindStruct("accounts", &accounts)
		accounts = append(accounts, reqData)
		conf.App.Set("accounts", accounts)
		conf.WriteAppConfig()

		ctx.JSON(iu.Resp{
			Result: true,
		})
	})

	app.Post("/account/remove", func(ctx iris.Context) {
		reqData := struct {
			ID string `json:"id"`
		}{}

		if !iu.ReadJSON(ctx, &reqData) {
			return
		}

		accounts := []g.Account{}
		conf.App.BindStruct("accounts", &accounts)

		filtered := []g.Account{}
		for i, a := range accounts {
			if a.ID != reqData.ID {
				filtered = append(filtered, accounts[i])
			}
		}

		conf.App.Set("accounts", filtered)
		conf.WriteAppConfig()

		ctx.JSON(iu.Resp{
			Result: true,
		})
	})

	app.Post("/account/fetch/one", func(ctx iris.Context) {
		reqData := struct {
			ID string `json:"id"`
		}{}

		if !iu.ReadJSON(ctx, &reqData) {
			return
		}

		accounts := []g.Account{}
		conf.App.BindStruct("accounts", &accounts)
		idx := -1

		for i, a := range accounts {
			if a.ID == reqData.ID {
				idx = i
				break
			}
		}

		if idx == -1 {
			golog.Errorf("No account with id %s found.", reqData.ID)
			ctx.JSON(iu.Resp{
				Result: false,
			})
			return
		}

		acc := accounts[idx]
		fetcher, err := fetcher.New(acc.Label, acc.ApiKey, acc.ApiSecret)
		if err != nil {
			golog.Error(err)
			ctx.JSON(iu.Resp{
				Result: false,
			})
			return
		}

		newFetch := time.Now().UTC()
		lastFetched, _ := time.Parse(time.RFC3339Nano, acc.LastFetched)

		err = fetcher.Ledger(lastFetched)
		if err != nil {
			golog.Error(err)
			ctx.JSON(iu.Resp{
				Result: false,
			})
			return
		}

		err = fetcher.Trades(lastFetched)
		if err != nil {
			golog.Error(err)
			ctx.JSON(iu.Resp{
				Result: false,
			})
			return
		}

		accounts[idx].LastFetched = newFetch.Format(time.RFC3339Nano)
		conf.App.Set("accounts", accounts)
		conf.WriteAppConfig()

		if err != nil {
			golog.Error(err)
		}
	})

	if err := app.Listen(address); err != nil {
		golog.Fatal(err)
	}
}

func registerFrontend(app *iris.Application, webAssets embed.FS) {
	var frontendTpl *view.HTMLEngine
	useEmbedded := conf.App.Bool("embedded")

	if useEmbedded {
		golog.Debug("Using embedded web sources")
		embeddedFs := iris.PrefixDir("frontend-dist", http.FS(webAssets))
		frontendTpl = iris.HTML(embeddedFs, ".html")
		app.HandleDir("/assets", embeddedFs)
	} else {
		golog.Debug("Using external web sources")
		frontendTpl = iris.HTML("./frontend-dist", ".html")
		app.HandleDir("/assets", "frontend-dist")
	}

	golog.Debug("Automatic reload of web sources is enabled")
	frontendTpl.Reload(conf.App.Bool("debug"))
	app.RegisterView(frontendTpl)
	app.OnAnyErrorCode(index)

	app.Get("/", index)
	app.Get("/{p:path}", index)
}

func index(ctx iris.Context) {
	ctx.View("index.html")
}
