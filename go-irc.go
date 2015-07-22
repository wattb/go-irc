package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net"
	"net/textproto"
	"os"
	"regexp"
	"strings"
	"time"
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

func NewBot(nick, server, port, owner, channel, pass string) *Bot {
	return &Bot{
		nick:    nick,
		server:  server,
		port:    port,
		channel: channel,
		pass:    pass,
		owner:   owner,
	}
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

// Set nick and authenticate with server
func (bot *Bot) Login() {
	fmt.Fprintf(bot.conn, "USER %s 8 * :%s\r\n", bot.nick, bot.nick)
	fmt.Fprintf(bot.conn, "NICK %s\r\n", bot.nick)
}

// Join a channel
func (bot *Bot) Join(channel string) {
	fmt.Fprintf(bot.conn, "JOIN %s\r\n", bot.channel)
}

func (bot *Bot) Respond(user *User, response string, msg *Message) {
	if msg.to == bot.nick {
		bot.write(fmt.Sprintf("PRIVMSG %s :%s\r\n", user.nick, response))
	} else {
		bot.write(fmt.Sprintf("PRIVMSG %s :%s: %s\r\n", msg.to, user.nick, response))
	}
}

// Write message to server
func (bot *Bot) write(msg string) {
	log.Printf("--> %s", msg)
	fmt.Fprintf(bot.conn, msg)
}

// Respond to PING with PONG + the random string
func (bot *Bot) Pong(pong *Message) {
	res := fmt.Sprintf("PONG :%s\r\n", pong.content)
	bot.write(res)
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

// Lookup a topic on wikipedia
func (bot *Bot) Wiki(args string) string {
	return fmt.Sprintf("https://en.wikipedia.org/w/index.php?search=%s&title=Special%%3ASearch&go=Go", args)
}

// Choose an item from a list
func (bot *Bot) Choose(args string) string {
	choices := strings.Split(args, ",")
	choice := choices[rand.Intn(len(choices))]

	return strings.TrimSpace(choice)
}

func (bot *Bot) Nick(nick string) {
	bot.nick = nick
	fmt.Fprintf(bot.conn, "NICK %s\r\n", bot.nick)
}

func (bot *Bot) Set(args string, user *User) string {
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
		bot.Nick(tar)
		return fmt.Sprintf("Nick set to %s.", bot.nick)
	case "owner":
		bot.owner = tar
		return fmt.Sprintf("Owner set to %s.", bot.owner)
	default:
		return "Unrecognised command! Options are: nick, owner."
	}
}

func shuffle(a []string) []string {
	for i := range a {
		j := rand.Intn(i + 1)
		a[i], a[j] = a[j], a[i]
	}
	return a
}

func (bot *Bot) Order(args string) string {
	choices := strings.Split(args, ",")
	trimmed := []string{}
	for _, w := range choices {
		trimmed = append(trimmed, strings.TrimSpace(w))
	}
	choices = shuffle(trimmed)
	ordered := strings.Join(choices, ", ")
	return ordered
}

func (bot *Bot) Markov(args string) string {
	return "This is supposed to be generated using a markov chain"
}

// Maps the user commands to the functions of the bot
func (bot *Bot) Command(msg *Message) {
	com, err1 := parseCommand(msg.content)
	user, err2 := parseSource(msg.source)
	if err1 == nil || err2 == nil {
		args := com.args
		res := ""
		switch com.command {
		case "wiki":
			res = bot.Wiki(args)
		case "c":
			res = bot.Choose(args)
		case "o":
			res = bot.Order(args)
		case "set":
			res = bot.Set(args, user)
		case "markov":
			res = bot.Markov(args)
		case "commands":
			res = "The available commands are: wiki, c, o, set, markov."
		default:
			res = "That isn't a command. Try .commands to see some."
		}
		log.Printf("<-- %s\n", com.command)
		bot.Respond(user, res, msg)
	} else {

		if err1 != nil {
			log.Println(err1)
		}
		if err2 != nil {
			log.Println(err2)
		}
	}

}

func markov_write(writer *bufio.Writer, words string) {
	writer.WriteString(fmt.Sprintf("%s\n", words))
	writer.Flush()
}

var nickFlag = flag.String("nick", "kobobot", "The username for the bot.")
var serverFlag = flag.String("server", "irc.rizon.net", "The IRC server to connect to.")
var portFlag = flag.String("port", "6667", "The port to connect to the server on.")
var ownerFlag = flag.String("owner", "nanago", "The owner of the bot.")
var chanFlag = flag.String("channel", "#kobobot", "The default channel to connect to.")
var passFlag = flag.String("password", "", "The password for the channel.")

func main() {
	flag.Parse()

	// Create a new bot and a socket to the desired server
	bot := NewBot(*nickFlag, *serverFlag, *portFlag, *ownerFlag, *chanFlag, *passFlag)
	log.Println(bot)
	conn, _ := bot.Connect()
	defer conn.Close()

	// The bot connects to the server with the designated username
	bot.Login()
	time.Sleep(time.Second * 5)
	// Joins the bot to the default channel
	bot.Join(bot.channel)

	// For markov chain records
	f, err := os.Create("/tmp/markov")
	if err != nil {
		panic(err)
	}
	defer f.Close()
	writer := bufio.NewWriter(f)

	// Read from the socket constantly
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
			log.Println("--- ", msg.source)
			log.Println("--- ", msg.command)
			log.Println("--- ", msg.to)
			log.Println("--- ", msg.content)

			switch {
			case msg.command == "PING":
				bot.Pong(msg)
			case msg.command == "PRIVMSG":
				bot.Command(msg)
			}
		}
	}
}
