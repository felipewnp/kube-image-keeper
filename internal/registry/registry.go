package registry

import (
	"errors"
	"net/http"
	"regexp"
	"strings"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/remote/transport"
)

var Endpoint = "cache-registry-service:5000"
var Protocol = "http://"

func imageExists(ref name.Reference, options ...remote.Option) (bool, error) {
	_, err := remote.Head(ref, options...)
	if err != nil {
		if err, ok := err.(*transport.Error); ok {
			if err.StatusCode == http.StatusNotFound {
				return false, nil
			}
		}
		return false, err
	}

	return true, nil
}

func getDestinationName(sourceName string) (string, error) {
	sourceRef, err := name.ParseReference(sourceName, name.Insecure)
	if err != nil {
		return "", err
	}
	fullname := strings.ReplaceAll(sourceRef.Name(), "index.docker.io", "docker.io")
	return Endpoint + "/" + fullname, nil
}

func DeleteImage(imageName string) error {
	destName, err := getDestinationName(imageName)
	if err != nil {
		return err
	}
	ref, err := name.ParseReference(destName, name.Insecure)
	if err != nil {
		return err
	}

	if exists, err := imageExists(ref); !exists || err != nil {
		return err
	}

	descriptor, err := remote.Head(ref)
	if err != nil {
		return err
	}

	digest, err := name.NewDigest(destName+"@"+descriptor.Digest.String(), name.Insecure)

	if err != nil {
		return err
	}

	return remote.Delete(digest)
}

func CacheImage(imageName string, keychain authn.Keychain) (bool, error) {
	destName, err := getDestinationName(imageName)
	if err != nil {
		return false, err
	}
	destRef, err := name.ParseReference(destName, name.Insecure)
	if err != nil {
		return false, err
	}
	sourceRef, err := name.ParseReference(imageName, name.Insecure)
	if err != nil {
		return false, err
	}

	exists, err := imageExists(destRef)
	if err != nil {
		return false, err
	}
	if exists {
		return false, nil
	}

	auth := remote.WithAuthFromKeychain(keychain)
	exists, err = imageExists(sourceRef, auth)
	if err != nil {
		return false, err
	}
	if !exists {
		return false, errors.New("could not find source image")
	}

	image, err := remote.Image(sourceRef, auth)
	if err != nil {
		return false, err
	}

	if err := remote.Write(destRef, image); err != nil {
		return false, err
	}

	if err := remote.Put(destRef, image); err != nil {
		return false, err
	}

	return true, nil
}

func SanitizeName(image string) string {
	nameRegex := regexp.MustCompile(`[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*`)
	return strings.Join(nameRegex.FindAllString(image, -1), "-")
}
