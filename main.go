package main

import (
	"context"
	"database/sql"
	"encoding/xml"
	"fmt"
	"github.com/SuperFes/gator/internal/database"
	"github.com/google/uuid"
	"html"
	"internal/config"
	"io"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"time"
)

import _ "github.com/lib/pq"

type state struct {
	cfg *config.Config
	db  *database.Queries
}

type command struct {
	name string
	args []string
}

type RSSFeed struct {
	Channel struct {
		Title       string    `xml:"title"`
		Link        string    `xml:"link"`
		Description string    `xml:"description"`
		Item        []RSSItem `xml:"item"`
	} `xml:"channel"`
}

type RSSItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate"`
}

type commands struct {
	list map[string]func(*state, command) error
}

func (cmds *commands) register(name string, f func(*state, command) error) {
	cmds.list[name] = f
}

func (cmds *commands) execute(st *state, cmd command) error {
	f, ok := cmds.list[cmd.name]

	if !ok {
		return fmt.Errorf("unknown command: %s", cmd.name)
	}

	return f(st, cmd)
}

func parseArgs() command {
	cmd := command{}

	if len(os.Args) == 1 {
		fmt.Println("no command provided")

		os.Exit(1)
	}

	cmd.name = os.Args[1]
	cmd.args = os.Args[2:]

	return cmd
}

func fetchFeed(ctx context.Context, feedURL string) (*RSSFeed, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, feedURL, nil)

	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "gator v0.0.0")

	res, err := http.DefaultClient.Do(req)

	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	doc, err := io.ReadAll(res.Body)

	if err != nil {
		return nil, err
	}

	var feed RSSFeed

	xml.Unmarshal(doc, &feed)

	feed.Channel.Title = html.UnescapeString(feed.Channel.Title)
	feed.Channel.Description = html.UnescapeString(feed.Channel.Description)

	for i := range feed.Channel.Item {
		feed.Channel.Item[i].Title = html.UnescapeString(feed.Channel.Item[i].Title)
		feed.Channel.Item[i].Description = html.UnescapeString(feed.Channel.Item[i].Description)
	}

	return &feed, nil
}

func middlewareLoggedIn(handler func(s *state, cmd command, user database.User) error) func(*state, command) error {
	return func(s *state, c command) error {
		user, err := s.db.GetUser(context.Background(), s.cfg.User)

		if err != nil {
			user = database.User{}
		}

		return handler(s, c, user)
	}
}

func handleLogin(s *state, c command, user database.User) error {
	if len(c.args) != 1 {
		return fmt.Errorf("usage: login <username>")
	}

	if user.Name == "" {
		return fmt.Errorf("user not found")
	}

	if !s.cfg.SetUser(c.args[0]) {
		return fmt.Errorf("failed to set user")
	}

	return nil
}

func handleRegister(s *state, c command, user database.User) error {
	if len(c.args) != 1 {
		return fmt.Errorf("usage: register <username>")
	}

	_, err := s.db.CreateUser(context.Background(), database.CreateUserParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Name:      c.args[0],
	})

	if err != nil {
		funcName := getFuncName()

		return fmt.Errorf("%s: %w", funcName, err)
	}

	if !s.cfg.SetUser(c.args[0]) {
		return fmt.Errorf("failed to set user")
	}

	return nil
}

func getFuncName() string {
	pc, _, _, _ := runtime.Caller(1)

	return runtime.FuncForPC(pc).Name()
}

func handleReset(s *state, c command, user database.User) error {
	err := s.db.DeleteFeeds(context.Background())

	if err != nil {
		funcName := getFuncName()

		return fmt.Errorf("%s: %w", funcName, err)
	}

	err = s.db.DeleteUsers(context.Background())

	if err != nil {
		funcName := getFuncName()

		return fmt.Errorf("%s: %w", funcName, err)
	}

	return nil
}

func handleUsers(s *state, c command, user database.User) error {
	users, err := s.db.GetUsers(context.Background())

	if err != nil {
		funcName := getFuncName()

		return fmt.Errorf("%s: %w", funcName, err)
	}

	for _, user := range users {
		if user.Name == s.cfg.User {
			fmt.Printf(" * %s (current)\n", user.Name)
		} else {
			fmt.Printf(" * %s\n", user.Name)
		}
	}

	return nil

}

func handleAgg(s *state, c command, user database.User) error {
	if len(c.args) != 1 {
		return fmt.Errorf("usage: agg <time_between_reqs>")
	}

	ctx := context.Background()
	funcName := getFuncName()

	following, err := s.db.GetFeedFollows(ctx, s.cfg.User)

	if err != nil {
		return fmt.Errorf("%s: %w", funcName, err)
	}

	timeBetweenReqs, err := time.ParseDuration(c.args[0])

	if err != nil {
		return fmt.Errorf("%s: %w", funcName, err)
	}

	ticker := time.NewTicker(timeBetweenReqs)

	for ; ; <-ticker.C {
		for _, follow := range following {
			feed, err := fetchFeed(ctx, follow.Url)

			if err != nil {
				return fmt.Errorf("%s: %w", funcName, err)
			}

			for _, item := range feed.Channel.Item {
				_, err = s.db.AddPost(ctx, database.AddPostParams{
					ID:          uuid.New(),
					CreatedAt:   time.Now(),
					UpdatedAt:   time.Now(),
					Title:       sql.NullString{String: item.Title, Valid: true},
					Description: sql.NullString{String: item.Description, Valid: true},
					FeedID:      follow.FeedID,
					UserID:      user.ID,
				})

				if err != nil {
					return fmt.Errorf("%s: %w", funcName, err)
				}
			}
		}
	}

	return nil
}

func main() {
	conf, err := config.Read()

	if err != nil {
		fmt.Println(err)

		return
	}

	st := state{cfg: &conf}

	pg, err := sql.Open("postgres", conf.DbURL)

	if err != nil {
		fmt.Println(err)

		os.Exit(1)
	}

	st.db = database.New(pg)

	cmds := commands{list: make(map[string]func(*state, command) error)}

	cmds.register("login", middlewareLoggedIn(handleLogin))
	cmds.register("register", middlewareLoggedIn(handleRegister))
	cmds.register("reset", middlewareLoggedIn(handleReset))
	cmds.register("users", middlewareLoggedIn(handleUsers))
	cmds.register("agg", middlewareLoggedIn(handleAgg))
	cmds.register("addfeed", middlewareLoggedIn(handleAddFeed))
	cmds.register("feeds", middlewareLoggedIn(handleFeeds))
	cmds.register("follow", middlewareLoggedIn(handleFollow))
	cmds.register("unfollow", middlewareLoggedIn(handleUnfollow))
	cmds.register("following", middlewareLoggedIn(handleFollowing))
	cmds.register("browse", middlewareLoggedIn(handleBrowse))

	cmds.register("help", func(s *state, c command) error {
		fmt.Println("commands:")
		fmt.Println(" * login <username>")
		fmt.Println(" * register <username>")
		fmt.Println(" * reset")
		fmt.Println(" * users")
		fmt.Println(" * agg <time_between_reqs>")
		fmt.Println(" * addfeed <name> <url>")
		fmt.Println(" * feeds")
		fmt.Println(" * follow <url>")
		fmt.Println(" * unfollow <url>")
		fmt.Println(" * following")
		fmt.Println(" * browse <limit=5>")

		return nil
	})

	cmd := parseArgs()

	err = cmds.execute(&st, cmd)

	if err != nil {
		fmt.Println(err)

		os.Exit(1)
	}
}

func handleBrowse(s *state, cmd command, user database.User) error {
	limit := 5

	if len(cmd.args) > 0 {
		l, err := strconv.Atoi(cmd.args[0])

		if err != nil {
			return fmt.Errorf("invalid limit")
		}

		limit = l
	}

	items, err := s.db.GetPosts(context.Background(), database.GetPostsParams{
		Limit:  int32(limit),
		UserID: user.ID,
		Offset: 0,
	})

	if err != nil {
		funcName := getFuncName()

		return fmt.Errorf("%s: %w", funcName, err)
	}

	for _, item := range items {
		fmt.Printf(" * %s - %s, id: %s\n", item.Title, item.Url, item.ID)
	}

	return nil
}

func handleFollowing(s *state, c command, user database.User) error {
	following, err := s.db.GetFeedFollows(context.Background(), s.cfg.User)

	if err != nil {
		funcName := getFuncName()

		return fmt.Errorf("%s: %w", funcName, err)
	}

	for _, follow := range following {
		fmt.Printf(" * %s - %s\n", follow.FeedName, follow.Url)
	}

	return nil
}

func handleUnfollow(s *state, c command, user database.User) error {
	if len(c.args) != 1 {
		return fmt.Errorf("usage: unfollow <url>")
	}

	feed, err := s.db.GetFeed(context.Background(), c.args[0])

	if err != nil {
		funcName := getFuncName()

		return fmt.Errorf("%s: %w", funcName, err)
	}

	if feed.Name == "" {
		return fmt.Errorf("feed not found")
	}

	err = s.db.DeleteFeedFollow(context.Background(), database.DeleteFeedFollowParams{
		FeedID: feed.ID,
		UserID: user.ID,
	})

	if err != nil {
		funcName := getFuncName()

		return fmt.Errorf("%s: %w", funcName, err)
	}

	return nil
}

func handleFollow(s *state, c command, user database.User) error {
	if len(c.args) != 1 {
		return fmt.Errorf("usage: follow <url>")
	}

	feed, err := s.db.GetFeed(context.Background(), c.args[0])

	if err != nil {
		funcName := getFuncName()

		return fmt.Errorf("%s: %w", funcName, err)
	}

	if feed.Name == "" {
		return fmt.Errorf("feed not found")
	}

	_, err = s.db.CreateFeedFollow(context.Background(), database.CreateFeedFollowParams{
		ID:     uuid.New(),
		UserID: user.ID,
		FeedID: feed.ID,
	})

	if err != nil {
		funcName := getFuncName()

		return fmt.Errorf("%s: %w", funcName, err)
	}

	return nil
}

func handleFeeds(s *state, c command, user database.User) error {
	feeds, err := s.db.GetFeeds(context.Background())

	if err != nil {
		funcName := getFuncName()

		return fmt.Errorf("%s: %w", funcName, err)
	}

	for _, feed := range feeds {
		fmt.Printf(" * %s [%s] (%s)\n", feed.Name, feed.Url, feed.Username)
	}

	return nil

}

func handleAddFeed(s *state, c command, user database.User) error {
	if len(c.args) != 2 {
		return fmt.Errorf("usage: addfeed <name> <url>")
	}

	feed, err := s.db.CreateFeed(context.Background(), database.CreateFeedParams{
		ID:     uuid.New(),
		Name:   c.args[0],
		Url:    c.args[1],
		UserID: user.ID,
	})

	if err != nil {
		funcName := getFuncName()

		return fmt.Errorf("%s: %w", funcName, err)
	}

	yes, err := s.db.CreateFeedFollow(context.Background(), database.CreateFeedFollowParams{
		ID:     uuid.New(),
		UserID: user.ID,
		FeedID: feed.ID,
	})

	if err != nil {
		funcName := getFuncName()

		return fmt.Errorf("%s: %w", funcName, err)
	}

	fmt.Println(yes)

	return nil
}
