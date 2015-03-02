package main

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"net"
	"net/textproto"
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
func join(conn net.Conn, ircbot *Bot) {
	fmt.Fprintf(conn, "USER %s 8 * :%s\r\n", ircbot.nick, ircbot.nick)
	fmt.Fprintf(conn, "NICK %s\r\n", ircbot.nick)
	fmt.Fprintf(conn, "JOIN %s\r\n", ircbot.channel)
}

func privmsg(conn net.Conn, target string, message string) {
	fmt.Fprintf(conn, "PRIVMSG %s %s\r\n", target, message)
}

// Check if line is PING
func ping(line string) bool {
	return strings.HasPrefix(line, "PING")
}

// Respond to PING with PONG + the random string
func pingResponse(conn net.Conn, pong *Message) {
	fmt.Fprintf(conn, "PONG %s\r\n", pong.content)
}

// Parse message source for nick, user and hostname values
func parseSource(source string) string {
	re, _ := regexp.Compile("^:(.+?)!(.*)@(.*)$")
	out := re.FindStringSubmatch(source)
	if out != nil {
		return &User{nick: out[1],
			name: out[2],
			host: out[3]}, nil
	} else {
		return &User{}, error("No user found in message source")
	}

}
func commands(conn net.Conn, msg *Message, bot *Bot) {
	user, err := parseSource(msg.source)
	if err != nil {
		log.Printf("Parsing is broken somewhere for %s", msg.source)
	}
	if user.nick == bot.owner {
		//
	}
}

func main() {
	bot := NewBot()
	conn, _ := bot.Connect()
	join(conn, bot)
	defer conn.Close()

	reader := bufio.NewReader(conn)
	tp := textproto.NewReader(reader)
	for {
		line, err := tp.ReadLine()
		if err != nil {
			log.Fatal("Error reading connection stream")
			break
		}
		fmt.Printf("%s\n", line)

		// Pack message into struct
		msg, err := parseLine(line)
		// Perform actions depending on the content of the message
		if err == nil {
			log.Printf("%s", msg.source)
			log.Printf("%s", msg.to)
			log.Printf("%s", msg.command)
			log.Printf("%s", msg.content)

			switch {
			case msg.command == "PING":
				pingResponse(conn, msg)
			case msg.command == "PRIVMSG":
				commands(conn, msg, bot)
			}
		}
	}
}
