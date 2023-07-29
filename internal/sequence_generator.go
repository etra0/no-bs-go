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

const IMAGE_DURATION = 4
const TRANSITION_DURATION = 0.25

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
func (slideshow *Slideshow) GenerateVideo() *string {
	firstPassVideo, _ := os.CreateTemp("", "*.mp4")
	firstPassVideo.Close()

	var args []string

	for _, image := range slideshow.Images {
		args = append(args, "-loop", "1", "-t", strconv.Itoa(IMAGE_DURATION), "-i", image)
	}

	args = append(args, "-filter_complex")
	// Build the filter complex pipeline.
	last_out := "[img0]"
	var filterBuilder strings.Builder
	// First, we resize all images to 1080x1920.
	for i := 0; i < len(slideshow.Images); i++ {
		filterBuilder.WriteString(fmt.Sprintf("[%d]scale=1080:1920:force_original_aspect_ratio=decrease,pad=1080:1920:-1:-1,setsar=1,format=yuv420p[img%d];", i, i))
	}

	// Then, we join all the images using the transition slideleft.
	for i := 1; i < len(slideshow.Images); i++ {
		filterBuilder.WriteString(fmt.Sprintf("%s[img%d]xfade=transition=slideleft:duration=%f:offset=%f[f%d];", last_out, i, TRANSITION_DURATION, (IMAGE_DURATION-TRANSITION_DURATION)*float32(i), i-1))
		last_out = fmt.Sprintf("[f%d]", i-1)
	}

	filter := strings.TrimRight(filterBuilder.String(), ";")
	args = append(args, filter, "-map", last_out, "-r", "60", "-pix_fmt", "yuv420p", "-c:v", "libx264", "-an", "-y", firstPassVideo.Name())

	log.Println("Generating first pass video...")
	err := exec.Command("ffmpeg", args...).Run()
	if err != nil {
		log.Println("Error generating first pass video: ", err)
		return nil
	}

	secondPassVideo, _ := os.CreateTemp("", "*.mp4")
	secondPassVideo.Close()

	// Now we have the first pass video. We need to add the audio. We need to choose between the audio and the
	// video, depending on which one is longer.
	longest := getLongestDuration(firstPassVideo.Name(), slideshow.Audio)
	var secondPassArgs []string
	switch longest {
	case "audio":
		// In cases where we only have two images, I find more useful to repeat the last frame for the rest of
		// the video since most of the time the second image is the relevant one.
		if len(slideshow.Images) == 2 {
			audioDuration := getDuration(slideshow.Audio) + 1
			secondPassArgs = append(secondPassArgs, "-i", firstPassVideo.Name(), "-i", slideshow.Audio, "-vf", fmt.Sprintf("tpad=stop_mode=clone:stop_duration=%f", audioDuration))
		} else {
			secondPassArgs = append(secondPassArgs, "-stream_loop", "-1", "-i", firstPassVideo.Name(), "-i", slideshow.Audio)
		}
		// In cases where the audio is longer than the video, we need to re-encode the video to match the audio.
		secondPassArgs = append(secondPassArgs, "-shortest", "-fflags", "shortest", "-max_interleave_delta", "100M", "-map", "0:v", "-map", "1:a", "-c:v", "libx264", "-y", secondPassVideo.Name())
	case "video":
		secondPassArgs = append(secondPassArgs, "-i", firstPassVideo.Name(), "-stream_loop", "-1", "-i", slideshow.Audio)
		// In the case the video is longer than the audio, we don't need to re-encode the video itself.
		secondPassArgs = append(secondPassArgs, "-shortest", "-fflags", "shortest", "-max_interleave_delta", "100M", "-map", "0:v", "-map", "1:a", "-c:v", "copy", "-y", secondPassVideo.Name())
	}
	// In here we use a lot of shenanigans to avoid some of the bufferings FFMPEG does, with the
	// max_interleave_delta we make sure to avoid extending the video more than we need.
	log.Println("Longest: ", longest, " Generating second pass video...")

	err = exec.Command("ffmpeg", secondPassArgs...).Run()
	if err != nil {
		log.Println("Error generating second pass video: ", err)
		return nil
	}

	secondPassVideoName := secondPassVideo.Name()
	log.Println("Second pass video generated successfully, ", secondPassVideoName)

	// Now we can cleanup the files.
	slideshow.Cleanup()
	os.Remove(firstPassVideo.Name())
	return &secondPassVideoName
}

// Remove both the images and the audio from the filesystem.
func (slideshow *Slideshow) Cleanup() {
	for _, image := range slideshow.Images {
		os.Remove(image)
	}
	os.Remove(slideshow.Audio)
}
