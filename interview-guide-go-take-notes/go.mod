module interview-guide-go

go 1.24.1

// 第三方库
require (
	github.com/aws/aws-sdk-go-v2 v1.41.5 // AWS SDK for Go
	github.com/aws/aws-sdk-go-v2/config v1.32.14 // AWS SDK for Go - Config
	github.com/aws/aws-sdk-go-v2/credentials v1.19.14 // AWS SDK for Go - Credentials
	github.com/aws/aws-sdk-go-v2/service/s3 v1.99.0 // AWS S3 服务
	github.com/go-chi/chi/v5 v5.2.5 // HTTP 路由
	github.com/jackc/pgx/v5 v5.7.4 // indirect; PostgreSQL 数据库驱动
	go.uber.org/zap v1.27.1 // 日志打印
	gorm.io/driver/postgres v1.6.0 // PostgreSQL 数据库驱动
	gorm.io/gorm v1.31.1 // GORM 数据库 ORM
)

// 间接依赖
require (
	github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream v1.7.8 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.18.21 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.4.21 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.7.21 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.8.6 // indirect
	github.com/aws/aws-sdk-go-v2/internal/v4a v1.4.22 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.13.7 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/checksum v1.9.13 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.13.21 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/s3shared v1.19.21 // indirect
	github.com/aws/aws-sdk-go-v2/service/signin v1.0.9 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.30.15 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.35.19 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.41.10 // indirect
	github.com/aws/smithy-go v1.24.2 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	go.uber.org/multierr v1.10.0 // indirect
	golang.org/x/crypto v0.32.0 // indirect
	golang.org/x/sync v0.10.0 // indirect
	golang.org/x/text v0.21.0 // indirect
)

require (
	github.com/google/uuid v1.6.0
	github.com/google/wire v0.7.0
	github.com/openai/openai-go v1.12.0
	github.com/redis/go-redis/v9 v9.18.0
	github.com/signintech/gopdf v0.36.0
	github.com/tsawler/tabula v1.6.6
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/otiai10/gosseract/v2 v2.4.1 // indirect
	github.com/phpdave11/gofpdi v1.0.14-0.20211212211723-1f10f9844311 // indirect
	github.com/pkg/errors v0.8.1 // indirect
	github.com/tidwall/gjson v1.14.4 // indirect
	github.com/tidwall/match v1.1.1 // indirect
	github.com/tidwall/pretty v1.2.1 // indirect
	github.com/tidwall/sjson v1.2.5 // indirect
	go.uber.org/atomic v1.11.0 // indirect
	golang.org/x/image v0.18.0 // indirect
	golang.org/x/net v0.34.0 // indirect
)
