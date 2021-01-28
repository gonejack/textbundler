package Textbundler

import (
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/gonejack/textbundler/util"
	"github.com/russross/blackfriday/v2"
	"github.com/schollz/progressbar/v3"
)

// Textbundle represents a textbundle for transferring Markdown files between
// applications, as defined in http://textbundle.org/.
type Textbundle struct {
	tempDir   string
	assetsDir string

	absMdPath          string
	processAttachments bool
	verbose            bool

	imgReplacements        map[string]string
	attachmentReplacements map[*blackfriday.LinkData]string
}

func (t *Textbundle) newAsset(filename string) (*os.File, error) {
	path := filepath.Join(t.assetsDir, filename)
	return os.Create(path)
}

func (t *Textbundle) visitor(node *blackfriday.Node, entering bool) blackfriday.WalkStatus {
	if node.Type == blackfriday.Image {
		ld := node.LinkData
		imageRef := string(ld.Destination)

		filename := util.GetFilename(imageRef)

		file, err := t.newAsset(filename)
		if err != nil {
			log.Fatal("Error creating asset file:", err)
		}
		defer file.Close()

		if util.IsValidURL(imageRef) {
			resp, err := http.Get(imageRef)
			if err != nil {
				log.Fatal("Error downloading image:", err)
			}
			defer resp.Body.Close()

			if t.verbose {
				bar := progressbar.NewOptions64(resp.ContentLength,
					progressbar.OptionSetTheme(progressbar.Theme{Saucer: "=", SaucerPadding: ".", BarStart: "|", BarEnd: "|"}),
					progressbar.OptionSetWidth(10),
					progressbar.OptionSpinnerType(11),
					progressbar.OptionShowBytes(true),
					progressbar.OptionShowCount(),
					progressbar.OptionSetPredictTime(false),
					progressbar.OptionSetDescription(imageRef),
					progressbar.OptionSetRenderBlankState(true),
					progressbar.OptionClearOnFinish(),
				)
				_ = bar.RenderBlank()
				_, err = io.Copy(io.MultiWriter(file, bar), resp.Body)
				_ = bar.Clear()
			} else {
				_, err = io.Copy(file, resp.Body)
			}

			if err != nil {
				log.Fatal("Error downloading image:", err)
			}
		} else {
			absImagePath := filepath.Join(filepath.Dir(t.absMdPath), imageRef)
			localImg, err := os.Open(absImagePath)
			if err != nil {
				log.Fatal("Error opening local image:", err)
			}
			defer localImg.Close()

			_, err = io.Copy(file, localImg)
			if err != nil {
				log.Fatal("Error copying image:", err)
			}
		}

		t.imgReplacements[imageRef] = filepath.Join("assets", filename)
	}

	if node.Type == blackfriday.Link && t.processAttachments {
		ref := string(node.LinkData.Destination)

		if !util.IsValidURL(ref) {
			filename := filepath.Base(ref)

			t.attachmentReplacements[&node.LinkData] = "#todo/process-attachment (" + filename + ")"
		}
	}

	return blackfriday.GoToNext
}

// NewTextbundle creates a new Textbundle, initiating a temporary directory for
// storing files during creation.
func NewTextbundle(absMdPath string, processAttachments, verbose bool) (*Textbundle, error) {
	t := new(Textbundle)
	t.imgReplacements = make(map[string]string)
	t.attachmentReplacements = make(map[*blackfriday.LinkData]string)

	t.absMdPath = absMdPath
	t.processAttachments = processAttachments
	t.verbose = verbose

	var err error
	t.tempDir, err = ioutil.TempDir("", "Textbundle")
	if err != nil {
		return nil, err
	}

	t.assetsDir = filepath.Join(t.tempDir, "assets")
	if err := os.Mkdir(t.assetsDir, os.ModePerm); err != nil {
		return nil, err
	}

	return t, nil
}

// GenerateBundle generates a Textbundle gives a Markdown file.
func GenerateBundle(mdContents []byte, absMdPath string, creation, modification time.Time, dest string, processAttachments, verbose bool, toAppend string) error {
	bundle, err := NewTextbundle(absMdPath, processAttachments, verbose)
	if err != nil {
		return err
	}

	processor := blackfriday.New()
	rootNode := processor.Parse(mdContents)
	rootNode.Walk(bundle.visitor)

	output := string(mdContents)

	for orig, replacement := range bundle.imgReplacements {
		output = strings.Replace(output, orig, replacement, -1)
	}

	for linkData, replacement := range bundle.attachmentReplacements {
		regex, err := regexp.Compile(`\[.*\].*\(.*` + string(linkData.Destination) + `.*\)`)
		if err != nil {
			return err
		}

		output = regex.ReplaceAllLiteralString(output, replacement)
	}

	if toAppend != "" {
		filename := filepath.Base(absMdPath)
		toAppendProcessed := strings.Replace(toAppend, `%f`, filename, -1)
		output = output + "\n" + toAppendProcessed + "\n"
	}

	err = ioutil.WriteFile(filepath.Join(bundle.tempDir, "text.markdown"), []byte(output), 0644)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(filepath.Join(bundle.tempDir, "info.json"), []byte(`
	{
		"transient" : true,
		"type" : "net.daringfireball.markdown",
		"creatorIdentifier" : "com.zachlatta.Textbundler",
		"version" : 2
	}
	`), 0644)
	if err != nil {
		return err
	}

	// Set creation and change time of the bundle so Bear knows when to mark the file as created / changed
	err = util.SetBirthTime(bundle.tempDir, creation)
	if err != nil {
		return err
	}
	err = util.SetModTime(bundle.tempDir, modification)
	if err != nil {
		return err
	}

	if filepath.Clean(dest) == filepath.Dir(dest) {
		filename := filepath.Base(absMdPath)
		err := os.Rename(bundle.tempDir, filepath.Join(dest, filename+".Textbundle"))
		if err != nil {
			return err
		}
	} else {
		err := os.Rename(bundle.tempDir, dest)
		if err != nil {
			return err
		}
	}

	return nil
}
