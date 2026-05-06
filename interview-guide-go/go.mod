module interview-guide-go

go 1.26.2

// 第三方库
require (
	github.com/aws/aws-sdk-go-v2 v1.41.7 // AWS SDK for Go
	github.com/aws/aws-sdk-go-v2/config v1.32.17 // AWS SDK for Go - Config
	github.com/aws/aws-sdk-go-v2/credentials v1.19.16 // AWS SDK for Go - Credentials
	github.com/aws/aws-sdk-go-v2/service/s3 v1.100.1 // AWS S3 服务
	github.com/go-chi/chi/v5 v5.2.5 // HTTP 路由
	github.com/jackc/pgx/v5 v5.9.2 // indirect; PostgreSQL 数据库驱动
	github.com/joho/godotenv v1.5.1 // 可选 ENV_FILE dotenv（docker run / 挂载配置）
	go.uber.org/zap v1.28.0 // 日志打印
	gorm.io/driver/postgres v1.6.0 // PostgreSQL 数据库驱动
	gorm.io/gorm v1.31.1 // GORM 数据库 ORM
)

// 间接依赖
require (
	github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream v1.7.10 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.18.23 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.4.23 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.7.23 // indirect
	github.com/aws/aws-sdk-go-v2/internal/v4a v1.4.24 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.13.9 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/checksum v1.9.15 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.13.23 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/s3shared v1.19.23 // indirect
	github.com/aws/aws-sdk-go-v2/service/signin v1.0.11 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.30.17 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.35.21 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.42.1 // indirect
	github.com/aws/smithy-go v1.25.1 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/sync v0.20.0 // indirect
	golang.org/x/text v0.36.0 // indirect
)

require (
	github.com/go-chi/cors v1.2.2
	github.com/google/uuid v1.6.0
	github.com/google/wire v0.7.0
	github.com/openai/openai-go v1.12.0
	github.com/pgvector/pgvector-go v0.3.0
	github.com/redis/go-redis/v9 v9.19.0
	github.com/signintech/gopdf v0.36.0
	github.com/tsawler/tabula v1.6.6
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/otiai10/gosseract/v2 v2.4.1 // indirect
	github.com/phpdave11/gofpdi v1.0.16 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/tidwall/gjson v1.18.0 // indirect
	github.com/tidwall/match v1.2.0 // indirect
	github.com/tidwall/pretty v1.2.1 // indirect
	github.com/tidwall/sjson v1.2.5 // indirect
	go.uber.org/atomic v1.11.0 // indirect
	golang.org/x/image v0.39.0 // indirect
	golang.org/x/net v0.53.0 // indirect
)
