module github.com/platinummonkey/remarkable-sync

go 1.25.0

require (
	github.com/google/uuid v1.6.0
	github.com/juruen/rmapi v0.0.32
	github.com/otiai10/gosseract/v2 v2.4.1
	github.com/pdfcpu/pdfcpu v0.11.1
	github.com/signintech/gopdf v0.34.0
	github.com/spf13/cobra v1.10.2
	github.com/spf13/viper v1.21.0
	go.uber.org/zap v1.27.1
)

require (
	github.com/adrg/strutil v0.3.1 // indirect
	github.com/adrg/sysfont v0.1.2 // indirect
	github.com/adrg/xdg v0.5.3 // indirect
	github.com/clipperhouse/stringish v0.1.1 // indirect
	github.com/clipperhouse/uax29/v2 v2.3.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/fsnotify/fsnotify v1.9.0 // indirect
	github.com/gabriel-vasile/mimetype v1.4.12 // indirect
	github.com/go-viper/mapstructure/v2 v2.4.0 // indirect
	github.com/golang-jwt/jwt v3.2.2+incompatible // indirect
	github.com/gorilla/i18n v0.0.0-20150820051429-8b358169da46 // indirect
	github.com/hhrutter/lzw v1.0.0 // indirect
	github.com/hhrutter/pkcs7 v0.2.0 // indirect
	github.com/hhrutter/tiff v1.0.2 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/mattn/go-runewidth v0.0.19 // indirect
	github.com/nfnt/resize v0.0.0-20180221191011-83c6a9932646 // indirect
	github.com/pelletier/go-toml/v2 v2.2.4 // indirect
	github.com/phpdave11/gofpdi v1.0.15 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/sagikazarmark/locafero v0.12.0 // indirect
	github.com/sirupsen/logrus v1.9.3 // indirect
	github.com/spf13/afero v1.15.0 // indirect
	github.com/spf13/cast v1.10.0 // indirect
	github.com/spf13/pflag v1.0.10 // indirect
	github.com/stretchr/testify v1.11.1 // indirect
	github.com/subosito/gotenv v1.6.0 // indirect
	github.com/unidoc/freetype v0.2.3 // indirect
	github.com/unidoc/pkcs7 v0.3.0 // indirect
	github.com/unidoc/timestamp v0.0.0-20200412005513-91597fd3793a // indirect
	github.com/unidoc/unichart v0.5.1 // indirect
	github.com/unidoc/unipdf/v3 v3.69.0 // indirect
	github.com/unidoc/unitype v0.5.1 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.yaml.in/yaml/v3 v3.0.4 // indirect
	golang.org/x/crypto v0.46.0 // indirect
	golang.org/x/image v0.34.0 // indirect
	golang.org/x/net v0.48.0 // indirect
	golang.org/x/sync v0.19.0 // indirect
	golang.org/x/sys v0.39.0 // indirect
	golang.org/x/text v0.32.0 // indirect
	golang.org/x/xerrors v0.0.0-20240903120638-7835f813f4da // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

// Use active fork maintained by ddvk instead of archived upstream
replace github.com/juruen/rmapi => github.com/ddvk/rmapi v0.0.32
