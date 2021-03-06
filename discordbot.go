package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"syscall"

	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/lus/dgc"
	"github.com/shopspring/decimal"
)

type Giphy struct {
	Data struct {
		Images struct {
			Original struct {
				URL string `json:"url"`
			} `json:"original"`
		} `json:"images"`
	} `json:"data"`
}

type Config struct {
	Token       string `json:"token"`
	GiphyAPIKey string `json:"giphyAPIKey"`
}

var (
	rMintNumber *regexp.Regexp
	rMintHash   *regexp.Regexp
	config      Config
)

func loadConfiguration(file string) (Config, error) {
	var c Config

	configFile, err := os.Open(file)
	if err != nil {
		return Config{}, fmt.Errorf("open: %w", err)
	}

	defer configFile.Close()
	if err != nil {
		return Config{}, fmt.Errorf("close: %w", err)
	}

	jsonParser := json.NewDecoder(configFile)
	err = jsonParser.Decode(&c)
	if err != nil {
		return Config{}, fmt.Errorf("decode: %w", err)
	}
	return c, nil
}

func startBot(configFile string) (*discordgo.Session, error) {
	log.Println("!-STARTING BOT-!")
	c, err := loadConfiguration(configFile)
	if err != nil {
		return nil, fmt.Errorf("invalid bot parameters: %v", err)
	}
	config = c
	log.Println(config.GiphyAPIKey)
	s, err := discordgo.New("Bot " + config.Token)
	if err != nil {
		return nil, fmt.Errorf("invalid bot parameters: %v", err)
	}
	err = s.Open()
	if err != nil {
		return nil, fmt.Errorf("error connecting: %v", err)
	}
	rMintNumber = regexp.MustCompile(`^([1-9]|[1-9][0-9]|[1-9][0-9][0-9]|[1-9][0-9][0-9][0-9]|10000)$`)
	rMintHash = regexp.MustCompile(`^\w{44}$`)
	log.Println("!-BOT STARTED-!")
	return s, nil
}

func registerCommands(r *dgc.Router) {
	r.RegisterCmd(&dgc.Command{
		Name:        "hello",
		Description: "Says hello",
		Usage:       "hello",
		Example:     "hello",
		IgnoreCase:  true,
		Handler: func(ctx *dgc.Ctx) {
			err := ctx.RespondText("Hello there!")
			if err != nil {
				log.Fatalf("RespondText: %v", err)
			}
		},
	})
	r.RegisterCmd(&dgc.Command{
		Name:        "robot",
		Description: "Random robot gif",
		Usage:       "robot",
		Example:     "robot fail",
		IgnoreCase:  true,
		Handler: func(ctx *dgc.Ctx) {
			reqUrl := fmt.Sprintf("https://api.giphy.com/v1/gifs/random?api_key=%s&tag=robot&rating=g", config.GiphyAPIKey)
			resp, err := http.Get(reqUrl)
			if err != nil {
				log.Fatalf("Get: %w", err)
			}
			respByte, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				log.Fatalf("ReadAll: %w", err)
			}
			log.Println(string(respByte))
			var gif Giphy
			err = json.Unmarshal(respByte, &gif)
			if err != nil {
				log.Fatalf("Unmarshal: %w", err)
			}
			err = ctx.RespondText(gif.Data.Images.Original.URL)
			if err != nil {
				log.Fatalf("RespondText: %w", err)
			}
		},
	})
	r.RegisterCmd(&dgc.Command{
		Name:        "solarian",
		Description: "get solarian by mint number",
		Usage:       "solarian",
		Example:     "solarian 1",
		IgnoreCase:  true,
		Handler: func(ctx *dgc.Ctx) {
			solarian := ctx.Arguments.Get(0).Raw()
			if rMintNumber.MatchString(solarian) || rMintHash.MatchString(solarian) { //match by mint number or mint hash
				resp, err := http.Get("http://dev1.solarians.click:8883/api/mints")
				if err != nil {
					log.Fatalf("request: %w", err)
				}
				byteResp, err := ioutil.ReadAll(resp.Body)
				if err != nil {
					log.Fatalf("ReadAll: %w", err)
				}
				mintInfo := []struct {
					Mint  string
					Parts []struct {
						Type      string
						Variation string
						Rarity    decimal.Decimal
					}
					TextAttributes []struct {
						Type      string
						Variation string
						Rarity    decimal.Decimal
					}
				}{}
				err = json.Unmarshal(byteResp, &mintInfo)
				if err != nil {
					log.Fatalf("Unmarshal: %w", err)
				}

				itemIndex := 99999
				for index, v := range mintInfo {
					if v.Mint == solarian || strings.Split(v.TextAttributes[0].Variation, "#")[1] == solarian {
						itemIndex = index
						break
					}
				}
				if itemIndex == 99999 {
					_, err := ctx.Session.ChannelMessageSendReply(ctx.Event.ChannelID, "Could not find solarian match", ctx.Event.Reference())
					if err != nil {
						log.Fatalf("solarian: %v", err)
					}
					return
				}
				embed := &discordgo.MessageEmbed{
					Fields: []*discordgo.MessageEmbedField{
						{
							Name:   "NAME",
							Value:  mintInfo[itemIndex].TextAttributes[1].Variation,
							Inline: true,
						},
						{
							Name:   "TITLE",
							Value:  mintInfo[itemIndex].TextAttributes[2].Variation + " | " + mintInfo[itemIndex].TextAttributes[2].Rarity.String() + "%",
							Inline: true,
						},
						{
							Name:   "LEVEL",
							Value:  mintInfo[itemIndex].TextAttributes[3].Variation + " | " + mintInfo[itemIndex].TextAttributes[3].Rarity.String() + "%",
							Inline: true,
						},
						{
							Name:   "LUCK",
							Value:  mintInfo[itemIndex].TextAttributes[4].Variation + " | " + mintInfo[itemIndex].TextAttributes[4].Rarity.String() + "%",
							Inline: true,
						},
						{
							Name:   "SCENE",
							Value:  mintInfo[itemIndex].Parts[0].Variation + " | " + mintInfo[itemIndex].Parts[0].Rarity.String() + "%",
							Inline: true,
						},
						{
							Name:   "LEGS",
							Value:  mintInfo[itemIndex].Parts[2].Variation + " | " + mintInfo[itemIndex].Parts[2].Rarity.String() + "%",
							Inline: true,
						},
						{
							Name:   "HANDS",
							Value:  mintInfo[itemIndex].Parts[3].Variation + " | " + mintInfo[itemIndex].Parts[3].Rarity.String() + "%",
							Inline: true,
						},
						{
							Name:   "TORSO",
							Value:  mintInfo[itemIndex].Parts[4].Variation + " | " + mintInfo[itemIndex].Parts[4].Rarity.String() + "%",
							Inline: true,
						},
						{
							Name:   "ANTENNA",
							Value:  mintInfo[itemIndex].Parts[5].Variation + " | " + mintInfo[itemIndex].Parts[5].Rarity.String() + "%",
							Inline: true,
						},
						{
							Name:   "HEAD",
							Value:  mintInfo[itemIndex].Parts[6].Variation + " | " + mintInfo[itemIndex].Parts[6].Rarity.String() + "%",
							Inline: true,
						},
						{
							Name:   "EYES",
							Value:  mintInfo[itemIndex].Parts[7].Variation + " | " + mintInfo[itemIndex].Parts[7].Rarity.String() + "%",
							Inline: true,
						},
						{
							Name:   "MOUTH",
							Value:  mintInfo[itemIndex].Parts[8].Variation + " | " + mintInfo[itemIndex].Parts[8].Rarity.String() + "%",
							Inline: true,
						},
					},
					Title: fmt.Sprintf(`Solarian %v`, mintInfo[itemIndex].TextAttributes[0].Variation),
					Type:  discordgo.EmbedTypeGifv,
					Image: &discordgo.MessageEmbedImage{
						URL:   "http://dev1.solarians.click:8883/render/" + mintInfo[itemIndex].Mint + ".gif",
						Width: 1000,
					},
				}
				_, err = ctx.Session.ChannelMessageSendComplex(ctx.Event.ChannelID, &discordgo.MessageSend{
					Embed:     embed,
					Reference: ctx.Event.Reference(),
				})
				if err != nil {
					log.Fatalf("solarian: %v", err)
				}
			} else {
				_, err := ctx.Session.ChannelMessageSendReply(ctx.Event.ChannelID, "That is not a valid mint number or mint hash", ctx.Event.Reference())
				if err != nil {
					log.Fatalf("solarian: %v", err)
				}
			}
		},
	})
}

func main() {
	session, err := startBot("config.json")
	if err != nil {
		panic(err)
	}
	router := dgc.Create(&dgc.Router{
		Prefixes: []string{"!"},
	})

	registerCommands(router)

	router.Initialize(session)

	defer func() {
		sc := make(chan os.Signal, 1)
		signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
		<-sc
	}()
}
