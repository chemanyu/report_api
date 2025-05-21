package logmodel

type Imp struct {
	ReqTime    string `json:"req_time"`
	UID        int    `json:"uid"`
	Ua         string `json:"ua"`
	Deviceid   string `json:"deviceid"`
	Raw        string `json:"raw"`
	URL        string `json:"url"`
	Referer    string `json:"referer"`
	Bower      int    `json:"bower"`
	TargetLink string `json:"target_link"`
	Cgid       int64  `json:"cgid"`
	Gid        int    `json:"gid"`
	UUID       string `json:"uuid"`
	CIP        string `json:"c_ip"`
	Region     string `json:"region"`
	Os         int    `json:"os"`
	IsNew      int    `json:"is_new"`
}

type Clk struct {
	ReqTime    string `json:"req_time"`
	UID        int    `json:"uid"`
	Ua         string `json:"ua"`
	Deviceid   string `json:"deviceid"`
	Raw        string `json:"raw"`
	URL        string `json:"url"`
	Referer    string `json:"referer"`
	Bower      int    `json:"bower"`
	TargetLink string `json:"target_link"`
	Cgid       int64  `json:"cgid"`
	Gid        int    `json:"gid"`
	UUID       string `json:"uuid"`
	CIP        string `json:"c_ip"`
	Region     string `json:"region"`
	Os         int    `json:"os"`
	IsNew      int    `json:"is_new"`
}

type Track struct {
	ReqTime     string  `json:"req_time"`
	UID         int     `json:"uid"`
	Ua          string  `json:"ua"`
	Deviceid    string  `json:"deviceid"`
	Raw         string  `json:"raw"`
	URL         string  `json:"url"`
	Referer     string  `json:"referer"`
	Bower       int     `json:"bower"`
	TargetLink  string  `json:"target_link"`
	Cgid        int64   `json:"cgid"`
	Gid         int     `json:"gid"`
	UUID        string  `json:"uuid"`
	CIP         string  `json:"c_ip"`
	Region      string  `json:"region"`
	Os          int     `json:"os"`
	IsNew       int     `json:"is_new"`
	Event       string  `json:"event"`
	Time        int     `json:"time"`
	User        string  `json:"user"`
	Loadtime    float64 `json:"loadtime"`
	Apptype     string  `json:"app_type"`
	LoddingPage string  `json:"loading_page"`
}
