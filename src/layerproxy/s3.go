package main

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"io/ioutil"
	"strings"
)

type S3Store struct {
	provider *s3.S3
}

func (store *S3Store) Get(key string) (*File, error) {
	parts := strings.Split(key, "/")
	if len(parts) <= 1 {
		return nil, ErrorDNE
	}
	bucket, keyPart := parts[0], strings.Join(parts[1:len(parts)], "/")
	if keyPart == "" {
		return nil, ErrorDNE
	}
	params := &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(keyPart),
	}
	resp, err := store.provider.GetObject(params)
	if err != nil {
		if strings.Contains(err.Error(), "NoSuchKey") || strings.Contains(err.Error(), "AccessDenied") {
			return nil, ErrorDNE
		}
		return nil, err
	}
	data, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return nil, err
	}
	file := &File{Key: key, Contents: string(data)}
	if resp.LastModified != nil {
		file.Mtime = *resp.LastModified
	}
	return file, nil
}

func (store *S3Store) Set(file *File) error {
	return nil
}

func (store *S3Store) List(prefix string) ([]*File, error) {
	parts := strings.Split(prefix, "/")
	var bucket string
	if len(parts) <= 1 {
		bucket, prefix = parts[0], ""
	} else {
		bucket, prefix = parts[0], strings.Join(parts[1:len(parts)], "/")
	}

	prefetchKeys := []*File{}
	params := &s3.ListObjectsInput{
		Bucket: aws.String(bucket),
		Prefix: aws.String(prefix),
	}
	err := store.provider.ListObjectsPages(params, func(page *s3.ListObjectsOutput, last bool) bool {
		for _, obj := range page.Contents {
			file := &File{Key: bucket + "/" + *obj.Key, Mtime: *obj.LastModified}
			prefetchKeys = append(prefetchKeys, file)

		}
		return true
	})
	if err != nil {
		log.Warning(fmt.Sprintf("Ran into err: %v", err))
		return nil, err
	}
	return prefetchKeys, nil

}

func (store *S3Store) Delete(key string) error {
	return nil
}

func NewS3Store(region string) *S3Store {
	creds := credentials.NewEnvCredentials()
	provider := s3.New(session.New(), &aws.Config{Region: aws.String(region), Credentials: creds})
	return &S3Store{provider: provider}
}
