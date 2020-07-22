package external

const (
	PORT             string = ":8081"
	UPLOAD_DIRECTORY string = "img"
	MAX_FILE_SIZE    int64  = 16 * 1024 * 1024
)

var AllowedExtensions []string = []string{
	"gif",
	"jpg",
	"jpeg",
	"png",
	"tiff",
}
