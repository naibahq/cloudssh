package router

import (
	"strings"

	"github.com/gofiber/fiber"
	"github.com/gofiber/logger"
	"golang.org/x/crypto/bcrypt"

	"github.com/naiba/cloudssh/cmd/server/dao"
	"github.com/naiba/cloudssh/internal/apiio"
	"github.com/naiba/cloudssh/internal/model"
	"github.com/naiba/cloudssh/pkg/validator"
	"github.com/naiba/cloudssh/pkg/xcrypto"
)

// Serve ..
func Serve(conf string, port int) {
	if err := dao.InitConfig(conf); err != nil {
		panic(err)
	}
	if err := dao.InitDB(dao.Conf.DBDSN); err != nil {
		panic(err)
	}
	if dao.Conf.Debug {
		dao.DB = dao.DB.Debug()
	}
	if err := dao.DB.AutoMigrate(&model.User{}, &model.Organization{}, &model.Host{}, &model.OrganizationMember{}, &model.HostOrganization{}).Error; err != nil {
		panic(err)
	}

	app := fiber.New()
	app.Use(logger.New())

	app.Use(func(c *fiber.Ctx) {
		authHeader := c.Get("Authorization")
		arr := strings.Split(authHeader, " ")
		if len(arr) == 2 {
			c.Locals("", "")
		}
		c.Next()
	})

	app.Post("/signup", func(c *fiber.Ctx) {
		var req apiio.RegisterRequest
		err := c.BodyParser(&req)
		if err == nil {
			err = validator.Validator.Struct(req)
		}
		if err == nil {
			_, err = xcrypto.BytesToPublicKey([]byte(req.Pubkey))
		}
		var user model.User
		if err == nil {
			user.Email = req.Email
			var ph []byte
			ph, err = bcrypt.GenerateFromPassword([]byte(req.PasswordHash), 14)
			user.PasswordHash = string(ph)
			user.EncryptKey = req.EncryptKey
			user.Privatekey = req.Privatekey
			user.Pubkey = req.Pubkey
		}
		if err == nil {
			err = user.RefreshToken()
		}
		if err == nil {
			err = dao.DB.Save(&user).Error
		}
		if err != nil {
			c.JSON(apiio.Response{
				Success: false,
				Message: err.Error(),
			})
			return
		}
		c.JSON(apiio.RegisterResponse{
			Response: apiio.Response{
				Success: true,
			},
			Data: user,
		})
	})
	app.Listen(port)
}

// CheckPasswordHash ..
func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
