package external

const (
	PORT             string = ":8081"
	MAX_FILE_SIZE    int    = 16 * 1024 * 1024
	UPLOAD_DIRECTORY string = "img"
)

var AllowedExtensions []string = []string{
	"gif",
	"jpg",
	"jpeg",
	"png",
	"tiff",
}
