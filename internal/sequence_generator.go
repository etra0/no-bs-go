package internal

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

type Slideshow struct {
	Images []string
	Audio  string
}

// This will build a *Slideshow from a request. It will download the images and audio.
// In case of failure of a single file, it'll give up and return an error.
func NewSlideshowFromRequest(rawJson map[string]interface{}) (*Slideshow, error) {
	image_urls := make([]string, 0)

	jsonImages := rawJson["picker"].([]interface{})
	for _, image := range jsonImages {
		var inner_image = image.(map[string]interface{})
		image_urls = append(image_urls, inner_image["url"].(string))
	}

	audio := rawJson["audio"].(string)

	images := make([]string, 0)
	for _, image := range image_urls {
		filepath, err := downloadFile(image, "*.jpg")
		if err != nil {
			return nil, err
		}

		images = append(images, filepath)
	}

	audio_file, err := downloadFile(audio, "*.mp3")
	if err != nil {
		return nil, err
	}

	return &Slideshow{Images: images, Audio: audio_file}, nil
}

func getDuration(file string) float32 {
	args := []string{"-v", "error", "-show_entries", "format=duration", "-of", "csv=p=0", file}
	cmd := exec.Command("ffprobe", args...)
	out, _ := cmd.Output()
	stringFloat := strings.TrimSpace(string(out))
	parsedValue, _ := strconv.ParseFloat(stringFloat, 32)
	log.Println("Duration: ", float32(parsedValue))
	return float32(parsedValue)
}

// This function returns either "video" or "audio" to choose between
// the media with the longest duration.
func getLongestDuration(video, audio string) string {
	videoDuration := getDuration(video)
	audioDuration := getDuration(audio)

	if audioDuration > videoDuration {
		return "audio"
	}

	return "video"
}

// This function builds the slideshow using FFmpeg. It returns the filepath of the output.
func (slideshow *Slideshow) GenerateVideo() string {
	firstPassVideo, _ := os.CreateTemp("", "*.mp4")
	firstPassVideo.Close()

	args := make([]string, 0)

	for _, image := range slideshow.Images {
		args = append(args, "-loop", "1", "-t", "3", "-i", image)
	}

	args = append(args, "-filter_complex")
	// Build the filter complex pipeline.
	last_out := "[img0]"
	filter := ""
	// First, we resize all images to 1080x1920.
	for i := 0; i < len(slideshow.Images); i++ {
		filter = fmt.Sprintf("%s;[%d]scale=1080:1920:force_original_aspect_ratio=decrease,pad=1080:1920:-1:-1,setsar=1,format=yuv420p[img%d]", filter, i, i)
	}

	// Then, we join all the images using the transition slideleft.
	for i := 1; i < len(slideshow.Images); i++ {
		filter = fmt.Sprintf("%s;%s[img%d]xfade=transition=slideleft:duration=0.5:offset=%f[f%d]", filter, last_out, i, 2.5*float32(i), i-1)
		last_out = fmt.Sprintf("[f%d]", i-1)
	}

	args = append(args, filter[1:], "-map", last_out, "-r", "25", "-pix_fmt", "yuv420p", "-c:v", "libx264", "-an", "-y", firstPassVideo.Name())

	log.Println("Generating first pass video...")
	exec.Command("ffmpeg", args...).Run()

	secondPassVideo, _ := os.CreateTemp("", "*.mp4")
	secondPassVideo.Close()

	// Now we have the first pass video. We need to add the audio.
	// We need to choose between the audio and the video, depending on which one is longer.
	longest := getLongestDuration(firstPassVideo.Name(), slideshow.Audio)
	var secondPassArgs []string
	switch longest {
	case "audio":
		// We need to add the audio to the video.
		secondPassArgs = append(secondPassArgs, "-i", firstPassVideo.Name(), "-stream_loop", "-1", "-i", slideshow.Audio, "-map", "0:v", "-map", "1:a", "-c:v", "copy", "-shortest", "-y", secondPassVideo.Name())
	case "video":
		secondPassArgs = append(secondPassArgs, "-stream_loop", "-1", "-i", firstPassVideo.Name(), "-i", slideshow.Audio, "-shortest", "-fflags", "shortest", "-max_interleave_delta", "100M", "-map", "0:v:0", "-map", "1:a:0", "-c:v", "libx264", "-y", secondPassVideo.Name())
	}
	log.Println("Longest: ", longest, " Generating second pass video...")
	exec.Command("ffmpeg", secondPassArgs...).Run()

	// Now we can cleanup the files.
	slideshow.Cleanup()
	os.Remove(firstPassVideo.Name())
	return secondPassVideo.Name()
}

// Remove both the images and the audio from the filesystem.
func (slideshow *Slideshow) Cleanup() {
	for _, image := range slideshow.Images {
		os.Remove(image)
	}
	os.Remove(slideshow.Audio)
}
