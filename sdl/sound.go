package sdl

import (
	"math/rand/v2"
	"sync/atomic"
	"unsafe"

	"github.com/Zyko0/go-sdl3/sdl"
	"github.com/jnb666/chip16/vm"
	log "github.com/sirupsen/logrus"
)

const (
	AudioRate  = 48000
	SampleSize = 2
	BufferSize = 256
)

var (
	lerpTable = [3][4][2]float64{
		{{0.0, 1.0}, {1.0, 0.0}, {0.0, -1.0}, {-1.0, 0.0}},  // triangle
		{{0.0, 0.5}, {0.5, 1.0}, {-1.0, -0.5}, {-0.5, 0.0}}, // sawtooth
		{{1.0, 1.0}, {1.0, 1.0}, {0.0, 0.0}, {0.0, 0.0}},    // pulse
	}
	attackMsec = [16]int{
		2, 8, 16, 24, 38, 56, 68, 80, 100, 250, 500, 800, 1000, 3000, 5000, 8000,
	}
	decayMsec = [16]int{
		6, 24, 48, 72, 114, 168, 204, 240, 300, 750, 1500, 2400, 3000, 9000, 15000, 24000,
	}
	releaseMsec = [16]int{
		6, 24, 48, 72, 114, 168, 204, 240, 300, 750, 1500, 2400, 3000, 9000, 15000, 24000,
	}
)

// Sound implements the vm.Sound interface
type Sound struct {
	stream         *sdl.AudioStream
	sampleIndex    int
	sampleTotal    int
	periodIndex    int
	periodTotal    int
	attackSamples  int
	decaySamples   int
	releaseSamples int
	sustainSamples int
	maxVolume      int
	volume         float64
	sustain        float64
	env            vm.Envelope
	waveform       vm.Waveform
	useEnvelope    bool
	sndUpdated     atomic.Bool
}

func (s *Sound) Init(volume int) {
	if volume < 0 {
		return
	}
	spec := &sdl.AudioSpec{
		Format:   sdl.AUDIO_S16,
		Channels: 1,
		Freq:     AudioRate,
	}
	s.stream = sdl.AUDIO_DEVICE_DEFAULT_PLAYBACK.OpenAudioDeviceStream(spec, sdl.NewAudioStreamCallback(s.soundCallback))
	if s.stream == nil {
		panic("error opening audio stream")
	}
	s.waveform = vm.Pulse
	s.maxVolume = min(volume, 255) << 7
	s.volume = s.volumeLevel(7)
}

// Play sound with given frequency for msec milliseconds or until StopSound is called if msec is zero.
func (s *Sound) StartSound(freq, msec int16, useEnvelope bool) {
	log.Debugf("start sound: tone=%d dur=%d vol=%.0f env=%v", freq, msec, s.volume, useEnvelope)
	if s.stream != nil {
		s.stream.PauseDevice()
		s.sampleIndex = 0
		if msec == 0 {
			s.sampleTotal = 1e10
		} else {
			s.sampleTotal = int(msec) * AudioRate / 1000
		}
		s.periodTotal = AudioRate / int(freq+1)
		s.useEnvelope = useEnvelope
		if useEnvelope {
			s.initEnvelope()
		}
		s.stream.ResumeDevice()
		s.sndUpdated.Store(true)
	}
}

func (s *Sound) StopSound() {
	log.Debug("stop sound")
	if s.stream != nil {
		s.stream.PauseDevice()
		s.sndUpdated.Store(true)
	}
}

func (s *Sound) SetSoundParams(typ, vol uint8, env vm.Envelope) {
	log.Debugf("set sound params: typ=%s vol=%d, env=%+v", vm.Waveform(typ), vol, env)
	if s.stream != nil {
		s.waveform = vm.Waveform(typ & 3)
		s.volume = s.volumeLevel(int(vol))
		s.sustain = s.volumeLevel(int(env.Sustain))
		s.env = env
	}
}

func (s *Sound) soundCallback(stream *sdl.AudioStream, neededBytes, totalBytes int32) {
	var buffer [BufferSize]int16
	needed := int(neededBytes / SampleSize)
	for needed > 0 && s.sampleIndex < s.sampleTotal {
		n := min(needed, s.sampleTotal-s.sampleIndex, BufferSize)
		s.generateSamples(buffer[:n])
		stream.PutData(asBytes(buffer[:n]))
		needed -= n
	}
}

func (s *Sound) generateSamples(buffer []int16) {
	waveform := vm.Pulse
	if s.useEnvelope {
		waveform = s.waveform
	}
	for ix := range buffer {
		s.periodIndex++
		if s.periodIndex >= s.periodTotal {
			s.periodIndex = 0
		}
		i := s.periodIndex
		t := s.periodTotal
		var sample float64
		if waveform < vm.Noise {
			v1 := lerpTable[waveform][4*i/t][0]
			v2 := lerpTable[waveform][4*i/t][1]
			w := float64(i%(t/4)) / float64(t/4)
			sample = lerp(v1, v2, w)
		} else if waveform == vm.Noise {
			sample = 2*rand.Float64() - 1
		} else {
			panic("invalid sound waveform type")
		}
		if s.useEnvelope {
			buffer[ix] = s.applyADSR(sample)
		} else {
			buffer[ix] = int16(sample * s.volume)
		}
		s.sampleIndex++
	}
}

/*
		  ________ _____ Vol
		 /\
		/  \______ _____ Sus Vol
	   /          \
	  /            \
	 /              \
	 +---+-+-----+--+
	 | A |D|  S  |R |
*/

func (s *Sound) initEnvelope() {
	s.attackSamples = AudioRate * attackMsec[s.env.Attack] / 1000
	s.decaySamples = AudioRate * decayMsec[s.env.Decay] / 1000
	s.releaseSamples = AudioRate * releaseMsec[s.env.Release] / 1000
	if s.sampleTotal < s.attackSamples {
		s.attackSamples = s.sampleTotal
		s.decaySamples = 0
	} else if s.sampleTotal < s.attackSamples+s.decaySamples {
		s.decaySamples = s.sampleTotal - s.attackSamples
	}
	s.sustainSamples = max(s.sampleTotal-s.attackSamples-s.decaySamples, 0)
	s.sampleTotal += s.releaseSamples
	log.Debugf("audio: A=%d D=%d S=%d R=%d TOTAL=%d",
		s.attackSamples, s.decaySamples, s.sustainSamples, s.releaseSamples, s.sampleTotal)
}

func (s *Sound) applyADSR(sample float64) int16 {
	var level float64
	i := s.sampleIndex
	if i < s.attackSamples {
		level = s.volume * float64(i) / float64(s.attackSamples)
	} else if i < s.attackSamples+s.decaySamples {
		level = s.sustain + (s.volume-s.sustain)*
			(1-float64(i-s.attackSamples)/float64(s.decaySamples))
	} else if i < s.attackSamples+s.decaySamples+s.sustainSamples {
		level = s.sustain
	} else {
		start := s.attackSamples + s.decaySamples + s.sustainSamples
		level = s.sustain * (1 - float64(i-start)/float64(s.releaseSamples))
	}
	return int16(sample * level)
}

// convert volume from 0..15 -> signed 16 bit volume level
func (s *Sound) volumeLevel(n int) float64 {
	level := float64(min(n, 15))
	return float64(s.maxVolume) / (2 * (16 - level))
}

func asBytes(s []int16) []uint8 {
	return unsafe.Slice((*uint8)(unsafe.Pointer(&s[0])), len(s)*2)
}

func lerp(v0, v1, t float64) float64 {
	return v0 + t*(v1-v0)
}
