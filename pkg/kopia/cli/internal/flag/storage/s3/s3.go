package s3

import (
	"strings"

	"github.com/kanisterio/kanister/pkg/log"
	"github.com/kanisterio/kanister/pkg/safecli"

	"github.com/kanisterio/kanister/pkg/kopia/cli/internal/command"
	"github.com/kanisterio/kanister/pkg/kopia/cli/internal/flag/storage/model"
)

// New returns a builder for the S3 subcommand storage.
func New(s model.StorageFlag) (*safecli.Builder, error) {
	endpoint := resolveS3Endpoint(s.Location.Endpoint(), s.GetLogger())
	prefix := model.GenerateFullRepoPath(s.Location.Prefix(), s.RepoPathPrefix)
	return command.NewCommandBuilder(command.S3,
		Region(s.Location.Region()),
		Bucket(s.Location.BucketName()),
		Endpoint(endpoint),
		Prefix(prefix),
		DisableTLS(s.Location.IsInsecureEndpoint()),
		DisableTLSVerify(s.Location.HasSkipSSLVerify()),
	)
}

// resolveS3Endpoint removes the trailing slash and
// protocol from provided endpoint and
// returns the absolute endpoint string.
func resolveS3Endpoint(endpoint string, logger log.Logger) string {
	if endpoint == "" {
		return ""
	}

	if strings.HasSuffix(endpoint, "/") {
		logger.Print("Removing trailing slashes from the endpoint")
		endpoint = strings.TrimRight(endpoint, "/")
	}

	sp := strings.SplitN(endpoint, "://", 2)
	if len(sp) > 1 {
		logger.Print("Removing leading protocol from the endpoint")
	}

	return sp[len(sp)-1]
}
