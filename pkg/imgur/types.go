package imgur

type Privacy string

const (
	PrivacyPublic Privacy = "public"
	PrivacyHidden Privacy = "hidden"
)

type BasicItem struct {
	AccountId  int    `json:"account_id"`
	AccountUrl string `json:"account_url"`
	Id         string `json:"id"`
	Link       string `json:"link"`
}

type Item struct {
	BasicItem
	CommentCount int      `json:"comment_count"`
	Datetime     int      `json:"datetime"`
	Description  string   `json:"description"`
	Downs        int      `json:"downs"`
	Favorite     bool     `json:"favorite"`
	InGallery    bool     `json:"in_gallery"`
	InMostViral  bool     `json:"in_most_viral"`
	IsAd         bool     `json:"is_ad"`
	IsAlbum      bool     `json:"is_album"`
	Nsfw         bool     `json:"nsfw"`
	Points       int      `json:"points"`
	Section      string   `json:"section"`
	Tags         []string `json:"tags"`
	Title        string   `json:"title"`
	Ups          int      `json:"ups"`
	Views        int      `json:"views"`
	Vote         string   `json:"vote"`
}

type Image struct {
	Item
	Animated   bool   `json:"animated"`
	HasSound   bool   `json:"has_sound"`
	Height     int    `json:"height"`
	Size       int    `json:"size"`
	Type       string `json:"type"`
	Width      int    `json:"width"`
	Link       string `json:"link"`
	Name       string `json:"name"`
	ParentId   string
	ParentName string
}

func (i *Image) Ratio() int {
	return (i.Width / i.Height) * 100
}

type Album struct {
	Item
	Cover       string  `json:"cover"`
	CoverHeight int     `json:"cover_height"`
	CoverWidth  int     `json:"cover_width"`
	Privacy     string  `json:"privacy"`
	Images      []Image `json:"images,omitempty"`
	ImagesCount int     `json:"images_count"`
}

type Folder struct {
	BasicItem
	Id          int    `json:"id"`
	Cover       string `json:"cover_hash"`
	CoverHeight int    `json:"cover_height"`
	CoverWidth  int    `json:"cover_width"`
	CreatedAt   string `json:"created_at"`
	Name        string `json:"name"`
	Link        string `json:"link"`
	UpdatedAt   string `json:"updated_at"`
}

// Convention seen on https://medium.com/random-go-tips/dynamic-json-schemas-part-1-8f7d103ace71
type ImageOrAlbum struct {
	Item
	*Image
	*Album
}

type APIResponse struct {
	Data    interface{} `json:"data"`
	Success bool        `json:"success"`
	Status  int         `json:"status"`
}

type ImagePostResponse struct {
	APIResponse
	Data Image `json:"data"`
}

type AlbumsResponse struct {
	APIResponse
	Data []Album `json:"data"`
}

type AlbumPostResponse struct {
	APIResponse
	Data Album `json:"data"`
}

type FoldersResponse struct {
	APIResponse
	Data []Folder `json:"data"`
}

type AlbumContentResponse struct {
	APIResponse
	Data []Image `json:"data"`
}

type FolderContentResponse struct {
	APIResponse
	Data []ImageOrAlbum `json:"data"`
}
