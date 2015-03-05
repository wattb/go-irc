package main

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net"
	"net/textproto"
	"os"
	"regexp"
	"strings"
)

type Message struct {
	source  string
	command string
	to      string
	content string
}

type User struct {
	nick string
	name string
	host string
}

type Com struct {
	command string
	args    string
}

type Bot struct {
	server  string
	port    string
	nick    string
	channel string
	pass    string
	owner   string
	conn    net.Conn
}

// Take a line and pack it into a struct representing a message
func parseLine(line string) (*Message, error) {
	re, _ := regexp.Compile(`^(.*?) ?([A-Z]+) ?(.*?) :(.+)`)
	msg := re.FindStringSubmatch(line)
	if msg != nil {
		return &Message{
			source:  msg[1],
			command: msg[2],
			to:      msg[3],
			content: msg[4]}, nil
	} else {
		return &Message{}, errors.New(fmt.Sprintf("No command in line %s", line))
	}

}

func NewBot() *Bot {
	return &Bot{
		server:  "irc.freenode.net",
		port:    "6667",
		nick:    "nanagonanashuu",
		channel: "#7l7wtest",
		pass:    "",
		owner:   "nanago",
		conn:    nil}
}

// Connect bot to IRC server
func (bot *Bot) Connect() (conn net.Conn, err error) {
	conn, err = net.Dial("tcp", bot.server+":"+bot.port)
	if err != nil {
		log.Fatal("Unable to connect to IRC server", err)
	}
	bot.conn = conn
	log.Printf("Connected to IRC server %s (%s)\n", bot.server, bot.conn.RemoteAddr())
	return bot.conn, nil
}

// Set nick, join channel
func join(bot *Bot) {
	fmt.Fprintf(bot.conn, "USER %s 8 * :%s\r\n", bot.nick, bot.nick)
	fmt.Fprintf(bot.conn, "NICK %s\r\n", bot.nick)
	fmt.Fprintf(bot.conn, "JOIN %s\r\n", bot.channel)
}

func respond(bot *Bot, user *User, response string, msg *Message) {
	res := ""
	if msg.to == bot.nick {
		res = fmt.Sprintf("PRIVMSG %s :%s\r\n", user.nick, response)
	} else {
		res = fmt.Sprintf("PRIVMSG %s :%s: %s\r\n", msg.to, user.nick, response)
	}
	log.Printf("--> %s", res)
	fmt.Fprintf(bot.conn, res)
}

// Check if line is PING
func ping(line string) bool {
	return strings.HasPrefix(line, "PING")
}

// Respond to PING with PONG + the random string
func pingResponse(bot *Bot, pong *Message) {
	res := fmt.Sprintf("PONG :%s\r\n", pong.content)
	log.Printf("--> %s", res)
	fmt.Fprintf(bot.conn, res)
}

// Parse message source for nick, user and hostname values
func parseSource(source string) (*User, error) {
	re, _ := regexp.Compile("^:(.+?)!(.*)@(.*)$")
	out := re.FindStringSubmatch(source)
	if out != nil {
		return &User{nick: out[1],
			name: out[2],
			host: out[3]}, nil
	} else {
		return &User{}, errors.New("No user found in message source")
	}

}

func parseCommand(command string) (*Com, error) {
	re, _ := regexp.Compile(`\.(\w+) ?(.*)$`)
	out := re.FindStringSubmatch(command)
	if out != nil {
		return &Com{
			command: out[1],
			args:    out[2]}, nil
	} else {
		return &Com{}, errors.New("Command could not be parsed")
	}
}

func wiki(bot *Bot, args string) string {
	return fmt.Sprintf("https://en.wikipedia.org/w/index.php?search=%s&title=Special%%3ASearch&go=Go", args)
}

func choose(bot *Bot, args string) string {
	choices := strings.Split(args, ",")
	return choices[rand.Intn(len(choices))]
}

func nick(bot *Bot, nick string) {
	bot.nick = nick
	fmt.Fprintf(bot.conn, "NICK %s\r\n", bot.nick)
}

func set(bot *Bot, args string, user *User) string {
	if user.nick != bot.owner {
		return "Only the bot owner can set values!"
	}
	in := strings.Split(args, " ")
	if len(args) < 2 {
		return "The set command requires two arguments!"
	}

	com := in[0]
	tar := in[1]
	switch com {
	case "nick":
		nick(bot, tar)
		return fmt.Sprintf("Nick set to %s.", bot.nick)
	case "owner":
		bot.owner = tar
		return fmt.Sprintf("Owner set to %s.", bot.owner)
	default:
		return "Unrecognised command! Options are: nick, owner."
	}
}

func shuffle(a []string) {
	for i := range a {
		j := rand.Intn(i + 1)
		a[i], a[j] = a[j], a[i]
	}
}

func order(bot *Bot, args string) string {
	args = strings.TrimSpace(args)
	choices := strings.Split(args, ",")
	shuffle(choices)
	ordered := strings.Join(choices, ", ")
	return ordered
}

func markov(bot *Bot, args string) string {
	return "This is supposed to be generated using a markov chain"
}

func commands(bot *Bot, msg *Message) {
	com, err1 := parseCommand(msg.content)
	user, err2 := parseSource(msg.source)
	if err1 == nil || err2 == nil {
		args := com.args
		res := ""
		switch com.command {
		case "wiki":
			res = wiki(bot, args)
		case "c":
			res = choose(bot, args)
		case "o":
			res = order(bot, args)
		case "set":
			res = set(bot, args, user)
		case "markov":
			res = markov(bot, args)
		case "commands":
			res = "The available commands are: wiki, c, o, set, markov."
		}
		if res != "" {
			respond(bot, user, res, msg)
		}
	}
}

func markov_write(writer *bufio.Writer, words string) {
	writer.WriteString(fmt.Sprintf("%s\n", words))
	writer.Flush()
}

func main() {

	bot := NewBot()
	conn, _ := bot.Connect()
	join(bot)
	defer conn.Close()

	f, err := os.Create("/tmp/markov")
	if err != nil {
		panic(err)
	}
	defer f.Close()
	writer := bufio.NewWriter(f)

	reader := bufio.NewReader(conn)
	tp := textproto.NewReader(reader)
	for {
		line, err := tp.ReadLine()
		if err != nil {
			log.Fatal("Error reading connection stream")
			break
		}
		log.Printf("<-- %s\n", line)

		// Pack message into struct
		msg, err := parseLine(line)
		if msg.to == bot.channel {
			go markov_write(writer, msg.content)
		}

		// Perform actions depending on the content of the message
		if err == nil {
			switch {
			case msg.command == "PING":
				pingResponse(bot, msg)
			case msg.command == "PRIVMSG":
				commands(bot, msg)
			}
		}
	}
}
