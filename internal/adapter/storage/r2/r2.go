package r2

import (
	"context"
	"fmt"
	"io"

	portssolicitacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/solicitacao"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

var _ portssolicitacao.Storage = (*Storage)(nil)

type Config struct {
	AccountID       string
	AccessKeyID     string
	SecretAccessKey string
	Bucket          string
}

type Storage struct {
	client *s3.Client
	bucket string
}

func Novo(ctx context.Context, cfg Config) (*Storage, error) {
	client, err := awsconfig.LoadDefaultConfig(ctx,
		awsconfig.WithRegion("auto"),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			cfg.AccessKeyID, cfg.SecretAccessKey, "",
		)),
	)
	if err != nil {
		return nil, err
	}

	return &Storage{
		client: s3.NewFromConfig(client, func(op *s3.Options) {
			op.BaseEndpoint = aws.String(fmt.Sprintf("https://%s.r2.cloudflarestorage.com", cfg.AccountID))
			op.UsePathStyle = true
		}),
		bucket: cfg.Bucket,
	}, nil
}

func (s *Storage) Enviar(ctx context.Context, chave string, conteudo io.Reader, contentType string) error {
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(chave),
		Body:        conteudo,
		ContentType: aws.String(contentType),
	})

	return err
}

func (s *Storage) Baixar(ctx context.Context, chave string) (io.ReadCloser, error) {
	resultado, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(chave),
	})
	if err != nil {
		return nil, err
	}

	return resultado.Body, nil
}

func (s *Storage) Remover(ctx context.Context, chave string) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(chave),
	})

	return err
}
