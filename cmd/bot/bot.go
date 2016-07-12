package main

import (
	"bytes"
	"encoding/binary"
	"flag"
	"fmt"
	"io"
	"math/rand"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/bwmarrin/discordgo"
	"github.com/dustin/go-humanize"
	redis "gopkg.in/redis.v3"
)

var (
	// discordgo session
	discord *discordgo.Session

	// Redis client connection (used for stats)
	rcli *redis.Client

	// Map of Guild id's to *Play channels, used for queuing and rate-limiting guilds
	queues map[string]chan *Play = make(map[string]chan *Play)

	// Sound encoding settings
	BITRATE        = 128
	MAX_QUEUE_SIZE = 6

	// Owner
	OWNER string

	// Shard (or -1)
	SHARDS []string = make([]string, 0)
)

// Play represents an individual use of the !airhorn command
type Play struct {
	GuildID   string
	ChannelID string
	UserID    string
	Sound     *Sound

	// The next play to occur after this, only used for chaining sounds like anotha
	Next *Play

	// If true, this was a forced play using a specific airhorn sound name
	Forced bool
}

type SoundCollection struct {
	Prefix    string
	Commands  []string
	Sounds    []*Sound
	ChainWith *SoundCollection

	soundRange int
}

type CommandCollection struct {
	Commands  []string
}


// Sound represents a sound clip
type Sound struct {
	Name string

	// Weight adjust how likely it is this song will play, higher = more likely
	Weight int

	// Delay (in milliseconds) for the bot to wait before sending the disconnect request
	PartDelay int

	// Buffer to store encoded PCM packets
	buffer [][]byte
}

// Array of all the sounds we have
var AIRHORN *SoundCollection = &SoundCollection{
	Prefix: "airhorn",
	Commands: []string{
		"!airhorn",
	},
	Sounds: []*Sound{
		createSound("default", 1000, 250),
		createSound("reverb", 800, 250),
		createSound("spam", 800, 0),
		createSound("tripletap", 800, 250),
		createSound("fourtap", 800, 250),
		createSound("distant", 500, 250),
		createSound("echo", 500, 250),
		createSound("clownfull", 250, 250),
		createSound("clownshort", 250, 250),
		createSound("clownspam", 250, 0),
		createSound("highfartlong", 200, 250),
		createSound("highfartshort", 200, 250),
		createSound("midshort", 100, 250),
		createSound("truck", 50, 250),
		createSound("spork", 25, 250),
	},
}

var ARFENHOUSE *SoundCollection = &SoundCollection{
	Prefix: "arfen",
	Commands: []string{
		"!arf",
	},
	Sounds: []*Sound{
		createSound("bulls", 100, 250),
		createSound("daddy", 100, 250),
		createSound("sex", 100, 250),
		createSound("amy", 100, 250),
		createSound("dead", 100, 250),
		createSound("bye", 100, 250),
		createSound("garbage", 100, 250),
		createSound("metabolife", 100, 250),
		createSound("whobez", 100, 250),
		createSound("burgers", 100, 250),
		createSound("insurance", 100, 250),
		createSound("joe", 100, 250),
		createSound("shutup", 100, 250),
		createSound("sun", 100, 250),
		createSound("sucks", 100, 250),
	},
}

var CENA *SoundCollection = &SoundCollection{
	Prefix: "jc",
	Commands: []string{
		"!johncena",
		"!cena",
	},
	Sounds: []*Sound{
		createSound("airhorn", 10, 250),
		createSound("birthday", 1, 250),
		createSound("echo", 10, 250),
		createSound("full", 10, 250),
		createSound("jc", 10, 250),
		createSound("nameis", 10, 250),
		createSound("spam", 10, 250),
	},
}

var COW *SoundCollection = &SoundCollection{
	Prefix: "cow",
	Commands: []string{
		"!cow",
	},
	Sounds: []*Sound{
		createSound("herd", 10, 250),
		createSound("moo", 10, 250),
		createSound("x3", 1, 250),
	},
}

var DECTALK *SoundCollection = &SoundCollection{
	Prefix: "dectalk",
	Commands: []string{
		"!dectalk",
		"!moonbase",
	},
	Sounds: []*Sound{
		createSound("ateam", 100, 250),
		createSound("batman", 100, 250),
		createSound("cena", 25, 250),
		createSound("choir", 100, 250),
		createSound("daisy", 25, 250),
		createSound("heya", 75, 250),
		createSound("imperial", 100, 250),
		createSound("mammamia", 100, 250),
		createSound("pizza", 50, 250),
		createSound("scooby", 100, 250),
		createSound("space", 100, 250),
		createSound("spooky", 100, 250),
		createSound("trolo", 25, 250),
		createSound("whalers", 75, 250),
	},
}

var DRWEIRD *SoundCollection = &SoundCollection{
	Prefix: "drweird",
	Commands: []string{
		"!drweird",
	},
	Sounds: []*Sound{
		createSound("different", 100, 250),
		createSound("fool", 100, 250),
		createSound("right", 100, 250),
	},
}

var DUNKED *SoundCollection = &SoundCollection{
	Prefix: "dunked",
	Commands: []string{
		"!dunk",
	},
	Sounds: []*Sound{
		createSound("getdunked", 1, 250),
	},
}

var EASPORTS *SoundCollection = &SoundCollection{
	Prefix: "easports",
	Commands: []string{
		"!easports",
	},
	Sounds: []*Sound{
		createSound("mad", 1, 250),
	},
}

var FREAK *SoundCollection = &SoundCollection{
	Prefix: "freak",
	Commands: []string{
		"!freak",
	},
	Sounds: []*Sound{
		createSound("cutitout", 100, 250),
		createSound("weenie", 100, 250),
	},
}

var JONTRON *SoundCollection = &SoundCollection{
	Prefix: "jontron",
	Commands: []string{
		"!jtron",
	},
	Sounds: []*Sound{
		createSound("brutal", 100, 250),
		createSound("halfchub", 100, 250),
		createSound("maze", 100, 250),
		createSound("notgonnawork", 40, 250),
		createSound("spaceace", 100, 250),
	},
}

var JURASSIC *SoundCollection = &SoundCollection{
	Prefix: "jurassic",
	Commands: []string{
		"!jurassic",
	},
	Sounds: []*Sound{
		createSound("melodica", 100, 250),
		createSound("password", 100, 250),
		createSound("wtf", 100, 250),
	},
}

var KATA *SoundCollection = &SoundCollection{
	Prefix: "kata",
	Commands: []string{
		"!kata",
	},
	Sounds: []*Sound{
		createSound("cosmos", 100, 250),
	},
}

var LOSING *SoundCollection = &SoundCollection{
	Prefix: "losing",
	Commands: []string{
		"!losing",
	},
	Sounds: []*Sound{
		createSound("tired", 10, 250),
	},
}

var MONKEY *SoundCollection = &SoundCollection{
	Prefix: "monkey",
	Commands: []string{
		"!monkey",
	},
	Sounds: []*Sound{
		createSound("business", 100, 250),
	},
}

var NEAT *SoundCollection = &SoundCollection{
	Prefix: "neat",
	Commands: []string{
		"!neat",
	},
	Sounds: []*Sound{
		createSound("howneat", 1, 250),
	},
}

var ORSON *SoundCollection = &SoundCollection{
	Prefix: "orson",
	Commands: []string{
		"!orson",
	},
	Sounds: []*Sound{
		createSound("anything", 1, 250),
	},
}

var PENIS *SoundCollection = &SoundCollection{
	Prefix: "penis",
	Commands: []string{
		"!penis",
	},
	Sounds: []*Sound{
		createSound("cockmaster", 50, 250),
		createSound("floppy", 100, 250),
		createSound("have", 100, 250),
		createSound("holy", 50, 250),
		createSound("jerk", 10, 250),
		createSound("kick", 100, 250),
		createSound("love", 100, 250),
		createSound("talk", 100, 250),
	},
}

var RANKUP *SoundCollection = &SoundCollection{
	Prefix: "rankup",
	Commands: []string{
		"!rankup",
	},
	Sounds: []*Sound{
		createSound("1", 100, 250),
		createSound("2", 50, 250),
		createSound("3", 25, 250),
		createSound("4", 12, 250),
		createSound("5", 6, 250),
	},
}

// Disabled because volume too loud right now.
//var SANIC *SoundCollection = &SoundCollection{
//	Prefix: "sanic",
//	Commands: []string{
//		"!sanic",		
//	},
//	Sounds: []*Sound{
//		createSound("damnit", 100, 250),
//		createSound("faster", 100, 250),
//		createSound("go", 100, 250),
//		createSound("jesus", 100, 250),
//		createSound("mph", 100, 250),
//		createSound("slow", 100, 250),
//	},
//}

var SBEMAIL *SoundCollection = &SoundCollection{
	Prefix: "sbemail",
	Commands: []string{
		"!sb",
	},
	Sounds: []*Sound{
		createSound("nine", 100, 250),
		createSound("trophy", 100, 250),
		createSound("steve", 100, 250),
		createSound("bus", 100, 250),
		createSound("beehan", 50, 250),
		createSound("face", 100, 250),
		createSound("perfect", 100, 250),
		createSound("money", 100, 250),
		createSound("ready", 100, 250),
		createSound("bye", 100, 250),
		createSound("thanks", 100, 250),
		createSound("jimmy", 50, 250),
	},
}

var SEALAB *SoundCollection = &SoundCollection{
	Prefix: "sealab",
	Commands: []string{
		"!sealab",
	},
	Sounds: []*Sound{
		createSound("awesome", 100, 250),
		createSound("booby", 100, 250),
		createSound("boss", 100, 250),
		createSound("fritters", 100, 250),
		createSound("getout", 100, 250),
		createSound("hurry", 100, 250),
		createSound("level", 100, 250),
		createSound("mindmeld", 100, 250),
		createSound("myclan", 100, 250),
		createSound("ragnor", 100, 250),
		createSound("shillelagh", 100, 250),
		createSound("shutup", 100, 250),
		createSound("smoothie", 100, 250),
		createSound("stick", 100, 250),
		createSound("teleport", 100, 250),
		createSound("why", 100, 250),
	},
}

var SIMPSONS *SoundCollection = &SoundCollection{
	Prefix: "simpsons",
	Commands: []string{
		"!simpsons",
	},
	Sounds: []*Sound{
		createSound("again", 100, 250),
		createSound("grandma", 100, 250),
		createSound("haha", 100, 250),
		createSound("no", 100, 250),
	},
}

var SNOOP *SoundCollection = &SoundCollection{
	Prefix: "snoop",
	Commands: []string{
		"!snoop",
	},
	Sounds: []*Sound{
		createSound("adventure", 100, 250),
		createSound("need", 100, 250),
		createSound("pokemon", 100, 250),
		createSound("punch", 100, 250),
		createSound("sm641", 100, 250),
		createSound("sm642", 100, 250),
		createSound("sm643", 100, 250),
		createSound("sm644", 100, 250),
		createSound("sm645", 100, 250),
		createSound("smb1", 100, 250),
		createSound("smb2", 100, 250),
		createSound("smb3", 100, 250),
		createSound("smb4", 100, 250),
		createSound("smb5", 100, 250),
		createSound("sml1", 100, 250),
		createSound("sml2", 100, 250),
		createSound("sml3", 100, 250),
		createSound("smokeverse", 100, 250),
		createSound("smw", 100, 250),
		createSound("sonic", 100, 250),
		createSound("storms", 100, 250),
		createSound("tales", 100, 250),
		createSound("thomas", 100, 250),
		createSound("weed", 100, 250),
	},
}

var SONGIFY *SoundCollection = &SoundCollection{
	Prefix: "songify",
	Commands: []string{
		"!songify",
	},
	Sounds: []*Sound{
		createSound("amen", 100, 250),
		createSound("balls", 25, 250),
		createSound("cat", 100, 250),
		createSound("dammit", 100, 250),
		createSound("dayum", 100, 250),
		createSound("dodges", 100, 250),
		createSound("father", 100, 250),
		createSound("females", 100, 250),
		createSound("hug", 100, 250),
		createSound("miracle", 100, 250),
		createSound("perfect", 100, 250),
		createSound("run", 100, 250),
		createSound("transition", 50, 250),
		createSound("wife", 10, 250),
	},
}

var SOUTHPARK *SoundCollection = &SoundCollection{
	Prefix: "sp",
	Commands: []string{
		"!sp",
		"!southpark",
	},
	Sounds: []*Sound{
		createSound("bathroom", 100, 250),
		createSound("diarrhea", 100, 250),
		createSound("finally", 100, 250),
		createSound("heroes", 100, 250),
		createSound("pos", 100, 250),
		createSound("pwnage", 100, 250),
		createSound("pwned", 100, 250),
	},
}

var WTF *SoundCollection = &SoundCollection{
	Prefix: "wtf",
	Commands: []string{
		"!wtf",
	},
	Sounds: []*Sound{
		createSound("disappointed", 100, 250),
		createSound("dumbass", 100, 250),
		createSound("happened", 100, 250),
		createSound("isthis", 100, 250),
		createSound("nobody", 100, 250),
		createSound("youdoing", 100, 250),
		createSound("yousaid", 25, 250),
	},
}

var CMDHELP *CommandCollection = &CommandCollection{
	Commands: []string{
		"!help",
		
	},
}

var CMDCOLORME *CommandCollection = &CommandCollection{
	Commands: []string{
		"!colorme",
		
	},
}

var COLLECTIONS []*SoundCollection = []*SoundCollection{
	AIRHORN,
	ARFENHOUSE,
	CENA,
	COW,
	DECTALK,
	DRWEIRD,
	DUNKED,
	EASPORTS,
	FREAK,
	JONTRON,
	JURASSIC,
	KATA,
	LOSING,
	MONKEY,
	NEAT,
	ORSON,
	PENIS,
	RANKUP,
	SBEMAIL,
	SEALAB,
	SIMPSONS,
	SNOOP,
	SONGIFY,
	SOUTHPARK,
	WTF,
}

var BOTCOMMANDS []*CommandCollection = []*CommandCollection{
	CMDHELP, CMDCOLORME,
}

// Create a Sound struct
func createSound(Name string, Weight int, PartDelay int) *Sound {
	return &Sound{
		Name:      Name,
		Weight:    Weight,
		PartDelay: PartDelay,
		buffer:    make([][]byte, 0),
	}
}

func (sc *SoundCollection) Load() {
	for _, sound := range sc.Sounds {
		sc.soundRange += sound.Weight
		sound.Load(sc)
	}
}

func (s *SoundCollection) Random() *Sound {
	var (
		i      int
		number int = randomRange(0, s.soundRange)
	)

	for _, sound := range s.Sounds {
		i += sound.Weight

		if number < i {
			return sound
		}
	}
	return nil
}

// Load attempts to load an encoded sound file from disk
// DCA files are pre-computed sound files that are easy to send to Discord.
// If you would like to create your own DCA files, please use:
// https://github.com/nstafie/dca-rs
// eg: dca-rs --raw -i <input wav file> > <output file>
func (s *Sound) Load(c *SoundCollection) error {
	path := fmt.Sprintf("audio/%v_%v.dca", c.Prefix, s.Name)

	file, err := os.Open(path)

	if err != nil {
		fmt.Println("error opening dca file :", err)
		return err
	}

	var opuslen int16

	for {
		// read opus frame length from dca file
		err = binary.Read(file, binary.LittleEndian, &opuslen)

		// If this is the end of the file, just return
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			return nil
		}

		if err != nil {
			fmt.Println("error reading from dca file :", err)
			return err
		}

		// read encoded pcm from dca file
		InBuf := make([]byte, opuslen)
		err = binary.Read(file, binary.LittleEndian, &InBuf)

		// Should not be any end of file errors
		if err != nil {
			fmt.Println("error reading from dca file :", err)
			return err
		}

		// append encoded pcm data to the buffer
		s.buffer = append(s.buffer, InBuf)
	}
}

// Plays this sound over the specified VoiceConnection
func (s *Sound) Play(vc *discordgo.VoiceConnection) {
	vc.Speaking(true)
	defer vc.Speaking(false)

	for _, buff := range s.buffer {
		vc.OpusSend <- buff
	}
}

// Attempts to find the current users voice channel inside a given guild
func getCurrentVoiceChannel(user *discordgo.User, guild *discordgo.Guild) *discordgo.Channel {
	for _, vs := range guild.VoiceStates {
		if vs.UserID == user.ID {
			channel, _ := discord.State.Channel(vs.ChannelID)
			return channel
		}
	}
	return nil
}

// Whether a guild id is in this shard
func shardContains(guildid string) bool {
	if len(SHARDS) != 0 {
		ok := false
		for _, shard := range SHARDS {
			if len(guildid) >= 5 && string(guildid[len(guildid)-5]) == shard {
				ok = true
				break
			}
		}
		return ok
	}
	return true
}

// Returns a random integer between min and max
func randomRange(min, max int) int {
	rand.Seed(time.Now().UTC().UnixNano())
	return rand.Intn(max-min) + min
}

// Prepares a play
func createPlay(user *discordgo.User, guild *discordgo.Guild, coll *SoundCollection, sound *Sound) *Play {
	// Grab the users voice channel
	channel := getCurrentVoiceChannel(user, guild)
	if channel == nil {
		log.WithFields(log.Fields{
			"user":  user.ID,
			"guild": guild.ID,
		}).Warning("Failed to find channel to play sound in")
		return nil
	}

	// Create the play
	play := &Play{
		GuildID:   guild.ID,
		ChannelID: channel.ID,
		UserID:    user.ID,
		Sound:     sound,
		Forced:    true,
	}

	// If we didn't get passed a manual sound, generate a random one
	if play.Sound == nil {
		play.Sound = coll.Random()
		play.Forced = false
	}

	// If the collection is a chained one, set the next sound
	if coll.ChainWith != nil {
		play.Next = &Play{
			GuildID:   play.GuildID,
			ChannelID: play.ChannelID,
			UserID:    play.UserID,
			Sound:     coll.ChainWith.Random(),
			Forced:    play.Forced,
		}
	}

	return play
}

// Prepares and enqueues a play into the ratelimit/buffer guild queue
func enqueuePlay(user *discordgo.User, guild *discordgo.Guild, coll *SoundCollection, sound *Sound) {
	play := createPlay(user, guild, coll, sound)
	if play == nil {
		return
	}

	// Check if we already have a connection to this guild
	//   yes, this isn't threadsafe, but its "OK" 99% of the time
	_, exists := queues[guild.ID]

	if exists {
		if len(queues[guild.ID]) < MAX_QUEUE_SIZE {
			queues[guild.ID] <- play
		}
	} else {
		queues[guild.ID] = make(chan *Play, MAX_QUEUE_SIZE)
		playSound(play, nil)
	}
}

func trackSoundStats(play *Play) {
	if rcli == nil {
		return
	}

	_, err := rcli.Pipelined(func(pipe *redis.Pipeline) error {
		var baseChar string

		if play.Forced {
			baseChar = "f"
		} else {
			baseChar = "a"
		}

		base := fmt.Sprintf("airhorn:%s", baseChar)
		pipe.Incr("airhorn:total")
		pipe.Incr(fmt.Sprintf("%s:total", base))
		pipe.Incr(fmt.Sprintf("%s:sound:%s", base, play.Sound.Name))
		pipe.Incr(fmt.Sprintf("%s:user:%s:sound:%s", base, play.UserID, play.Sound.Name))
		pipe.Incr(fmt.Sprintf("%s:guild:%s:sound:%s", base, play.GuildID, play.Sound.Name))
		pipe.Incr(fmt.Sprintf("%s:guild:%s:chan:%s:sound:%s", base, play.GuildID, play.ChannelID, play.Sound.Name))
		pipe.SAdd(fmt.Sprintf("%s:users", base), play.UserID)
		pipe.SAdd(fmt.Sprintf("%s:guilds", base), play.GuildID)
		pipe.SAdd(fmt.Sprintf("%s:channels", base), play.ChannelID)
		return nil
	})

	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Warning("Failed to track stats in redis")
	}
}

// Play a sound
func playSound(play *Play, vc *discordgo.VoiceConnection) (err error) {
	log.WithFields(log.Fields{
		"play": play,
	}).Info("Playing sound")

	if vc == nil {
		vc, err = discord.ChannelVoiceJoin(play.GuildID, play.ChannelID, false, false)
		// vc.Receive = false
		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
			}).Error("Failed to play sound")
			delete(queues, play.GuildID)
			return err
		}
	}

	// If we need to change channels, do that now
	if vc.ChannelID != play.ChannelID {
		vc.ChangeChannel(play.ChannelID, false, false)
		time.Sleep(time.Millisecond * 125)
	}

	// Track stats for this play in redis
	go trackSoundStats(play)

	// Sleep for a specified amount of time before playing the sound
	time.Sleep(time.Millisecond * 32)

	// Play the sound
	play.Sound.Play(vc)

	// If this is chained, play the chained sound
	if play.Next != nil {
		playSound(play.Next, vc)
	}

	// If there is another song in the queue, recurse and play that
	if len(queues[play.GuildID]) > 0 {
		play := <-queues[play.GuildID]
		playSound(play, vc)
		return nil
	}

	// If the queue is empty, delete it
	time.Sleep(time.Millisecond * time.Duration(play.Sound.PartDelay))
	delete(queues, play.GuildID)
	vc.Disconnect()
	return nil
}

func onReady(s *discordgo.Session, event *discordgo.Ready) {
	log.Info("Recieved READY payload")
	s.UpdateStatus(0, "dank memes")
}

func onGuildCreate(s *discordgo.Session, event *discordgo.GuildCreate) {
	if !shardContains(event.Guild.ID) {
		return
	}

	if event.Guild.Unavailable != nil {
		return
	}

	for _, channel := range event.Guild.Channels {
		if channel.ID == event.Guild.ID {
			s.ChannelMessageSend(channel.ID, "**AIRHORN BOT READY FOR HORNING. TYPE `!AIRHORN` WHILE IN A VOICE CHANNEL TO ACTIVATE**")
			return
		}
	}
}

func scontains(key string, options ...string) bool {
	for _, item := range options {
		if item == key {
			return true
		}
	}
	return false
}

func calculateAirhornsPerSecond(cid string) {
	current, _ := strconv.Atoi(rcli.Get("airhorn:a:total").Val())
	time.Sleep(time.Second * 10)
	latest, _ := strconv.Atoi(rcli.Get("airhorn:a:total").Val())

	discord.ChannelMessageSend(cid, fmt.Sprintf("Current APS: %v", (float64(latest-current))/10.0))
}

func displayBotStats(cid string) {
	stats := runtime.MemStats{}
	runtime.ReadMemStats(&stats)

	users := 0
	for _, guild := range discord.State.Ready.Guilds {
		users += len(guild.Members)
	}

	w := &tabwriter.Writer{}
	buf := &bytes.Buffer{}

	w.Init(buf, 0, 4, 0, ' ', 0)
	fmt.Fprintf(w, "```\n")
	fmt.Fprintf(w, "Discordgo: \t%s\n", discordgo.VERSION)
	fmt.Fprintf(w, "Go: \t%s\n", runtime.Version())
	fmt.Fprintf(w, "Memory: \t%s / %s (%s total allocated)\n", humanize.Bytes(stats.Alloc), humanize.Bytes(stats.Sys), humanize.Bytes(stats.TotalAlloc))
	fmt.Fprintf(w, "Tasks: \t%d\n", runtime.NumGoroutine())
	fmt.Fprintf(w, "Servers: \t%d\n", len(discord.State.Ready.Guilds))
	fmt.Fprintf(w, "Users: \t%d\n", users)
	fmt.Fprintf(w, "Shards: \t%s\n", strings.Join(SHARDS, ", "))
	fmt.Fprintf(w, "```\n")
	w.Flush()
	discord.ChannelMessageSend(cid, buf.String())
}

func utilSumRedisKeys(keys []string) int {
	results := make([]*redis.StringCmd, 0)

	rcli.Pipelined(func(pipe *redis.Pipeline) error {
		for _, key := range keys {
			results = append(results, pipe.Get(key))
		}
		return nil
	})

	var total int
	for _, i := range results {
		t, _ := strconv.Atoi(i.Val())
		total += t
	}

	return total
}

func displayUserStats(cid, uid string) {
	keys, err := rcli.Keys(fmt.Sprintf("airhorn:*:user:%s:sound:*", uid)).Result()
	if err != nil {
		return
	}

	totalAirhorns := utilSumRedisKeys(keys)
	discord.ChannelMessageSend(cid, fmt.Sprintf("Total Airhorns: %v", totalAirhorns))
}

func displayServerStats(cid, sid string) {
	keys, err := rcli.Keys(fmt.Sprintf("airhorn:*:guild:%s:sound:*", sid)).Result()
	if err != nil {
		return
	}

	totalAirhorns := utilSumRedisKeys(keys)
	discord.ChannelMessageSend(cid, fmt.Sprintf("Total Airhorns: %v", totalAirhorns))
}

func utilGetMentioned(s *discordgo.Session, m *discordgo.MessageCreate) *discordgo.User {
	for _, mention := range m.Mentions {
		if mention.ID != s.State.Ready.User.ID {
			return mention
		}
	}
	return nil
}

func airhornBomb(cid string, guild *discordgo.Guild, user *discordgo.User, cs string) {
	count, _ := strconv.Atoi(cs)
	discord.ChannelMessageSend(cid, ":ok_hand:"+strings.Repeat(":trumpet:", count))

	// Cap it at something
	if count > 100 {
		return
	}

	play := createPlay(user, guild, AIRHORN, nil)
	vc, err := discord.ChannelVoiceJoin(play.GuildID, play.ChannelID, true, true)
	if err != nil {
		return
	}

	for i := 0; i < count; i++ {
		AIRHORN.Random().Play(vc)
	}

	vc.Disconnect()
}

// Handles bot operator messages, should be refactored (lmao)
func handleBotControlMessages(s *discordgo.Session, m *discordgo.MessageCreate, parts []string, g *discordgo.Guild) {
	ourShard := shardContains(g.ID)

	if scontains(parts[1], "status") && ourShard {
		displayBotStats(m.ChannelID)
	} else if scontains(parts[1], "stats") && ourShard {
		if len(m.Mentions) >= 2 {
			displayUserStats(m.ChannelID, utilGetMentioned(s, m).ID)
		} else if len(parts) >= 3 {
			displayUserStats(m.ChannelID, parts[2])
		} else {
			displayServerStats(m.ChannelID, g.ID)
		}
	} else if scontains(parts[1], "bomb") && len(parts) >= 4 && ourShard {
		airhornBomb(m.ChannelID, g, utilGetMentioned(s, m), parts[3])
	} else if scontains(parts[1], "shards") {
		guilds := 0
		for _, guild := range s.State.Ready.Guilds {
			if shardContains(guild.ID) {
				guilds += 1
			}
		}
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf(
			"Shard %v contains %v servers",
			strings.Join(SHARDS, ","),
			guilds))
	} else if scontains(parts[1], "aps") && ourShard {
		s.ChannelMessageSend(m.ChannelID, ":ok_hand: give me a sec m8")
		go calculateAirhornsPerSecond(m.ChannelID)
	}
	return
}

func onMessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if len(m.Content) <= 0 || (m.Content[0] != '!' && len(m.Mentions) < 1) {
		return
	}

	msg := strings.Replace(m.ContentWithMentionsReplaced(), s.State.Ready.User.Username, "username", 1)
	parts := strings.Split(strings.ToLower(msg), " ")

	channel, _ := discord.State.Channel(m.ChannelID)
	if channel == nil {
		log.WithFields(log.Fields{
			"channel": m.ChannelID,
			"message": m.ID,
		}).Warning("Failed to grab channel")
		return
	}

	guild, _ := discord.State.Guild(channel.GuildID)
	if guild == nil {
		log.WithFields(log.Fields{
			"guild":   channel.GuildID,
			"channel": channel,
			"message": m.ID,
		}).Warning("Failed to grab guild")
		return
	}

	// If this is a mention, it should come from the owner (otherwise we don't care)
	if len(m.Mentions) > 0 && m.Author.ID == OWNER && len(parts) > 0 {
		mentioned := false
		for _, mention := range m.Mentions {
			mentioned = (mention.ID == s.State.Ready.User.ID)
			if mentioned {
				break
			}
		}

		if mentioned {
			handleBotControlMessages(s, m, parts, guild)
		}
		return
	}

	// If it's not relevant to our shard, just exit
	if !shardContains(guild.ID) {
		return
	}

	// Find the collection for the command we got
	for _, coll := range COLLECTIONS {
		if scontains(parts[0], coll.Commands...) {

			// If they passed a specific sound effect, find and select that (otherwise play nothing)
			var sound *Sound
			if len(parts) > 1 {
				for _, s := range coll.Sounds {
					if parts[1] == s.Name {
						sound = s
					}
				}

				if sound == nil {
					return
				}
			}

			go enqueuePlay(m.Author, guild, coll, sound)
			return
		}
	}
	
	
	// Check if message was a bot command instead of a sound
	for _, coll := range BOTCOMMANDS {
		if scontains(parts[0], coll.Commands...) {

			if parts[0] == "!help" {
				
			
			
				// If message passed a level 2 command, find and select that
				if len(parts) > 1 {
				
					cmdfound := false
				
					for _, coll2 := range COLLECTIONS {
						if scontains("!" + parts[1], coll2.Commands...) {
						
							helplist := "\n"
							cmdfound = true
						
							for _, j := range coll2.Sounds {
								helplist = helplist + j.Name + "\n"
							}
							s.ChannelMessageSend(m.ChannelID, "!" + parts[1] + " <sound>\n" + helplist)
						}
					}
					
					if !cmdfound {
						s.ChannelMessageSend(m.ChannelID, "Bro, that's not a thing. ¯\\_(ツ)_/¯")
					} 
				
					
				} else {
				
					// give level 1 help
					
					helplist := "I can play sounds from these categories.\n"
					
					for _, j := range COLLECTIONS {
					
						helplist = helplist + "\n"
					
						for _, j2 := range j.Commands {
							helplist = helplist + j2 + ", "
						}
					
					}
					
					helplist = helplist + "\n\nTry !help <category> for specific sounds."
					helplist = helplist + "\nIf you'd like to contribute to Droppy, please use the to-do spreadsheet: https://docs.google.com/spreadsheets/d/1hKDArZS85DQ2cQ3tVGHk_YIYHpsKXM6XHxdsas14-6s/edit#gid=0"
					s.ChannelMessageSend(m.ChannelID, helplist)
									
				}
				
			} else if parts[0] == "!colorme" {
			
				s.ChannelMessageSend(m.ChannelID, "Coming soon :)")
			
		}
	}
	
}

func main() {
	var (
		Token = flag.String("t", "", "Discord Authentication Token")
		Redis = flag.String("r", "", "Redis Connection String")
		Shard = flag.String("s", "", "Integers to shard by")
		Owner = flag.String("o", "", "Owner ID")
		err   error
	)
	flag.Parse()

	if *Owner != "" {
		OWNER = *Owner
	}

	// Make sure shard is either empty, or an integer
	if *Shard != "" {
		SHARDS = strings.Split(*Shard, ",")

		for _, shard := range SHARDS {
			if _, err := strconv.Atoi(shard); err != nil {
				log.WithFields(log.Fields{
					"shard": shard,
					"error": err,
				}).Fatal("Invalid Shard")
				return
			}
		}
	}

	// Preload all the sounds
	log.Info("Preloading sounds...")
	for _, coll := range COLLECTIONS {
		coll.Load()
	}

	// If we got passed a redis server, try to connect
	if *Redis != "" {
		log.Info("Connecting to redis...")
		rcli = redis.NewClient(&redis.Options{Addr: *Redis, DB: 0})
		_, err = rcli.Ping().Result()

		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
			}).Fatal("Failed to connect to redis")
			return
		}
	}

	// Create a discord session
	log.Info("Starting discord session...")
	discord, err = discordgo.New(*Token)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Fatal("Failed to create discord session")
		return
	}

	discord.AddHandler(onReady)
	discord.AddHandler(onGuildCreate)
	discord.AddHandler(onMessageCreate)

	err = discord.Open()
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Fatal("Failed to create discord websocket connection")
		return
	}

	// We're running!
	log.Info("Dropbot is ready to drop some dank memes.")

	// Wait for a signal to quit
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, os.Kill)
	<-c
}
