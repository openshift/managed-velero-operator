package gcs

import (
	"bytes"
	"context"
	"fmt"
	"testing"

	"cloud.google.com/go/storage"
	"github.com/googleapis/google-cloud-go-testing/storage/stiface"
	"k8s.io/apimachinery/pkg/runtime"
	fakekubeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestCreateBucket(t *testing.T) {

	ctx := context.Background()

	fakeGClient := newFakeClient()

	localObjects := []runtime.Object{}

	drv := &driver{
		Config: &GCS{
			Region:    "us-east1",
			Project:   "dummy-project-id",
			InfraName: "dummy-infra",
		},
		Context:    ctx,
		kubeClient: fakekubeclient.NewFakeClient(localObjects...),
	}
	err := drv.createBucket(fakeGClient, "dummy-bucket-name")
	if err != nil {
		t.Errorf("CreateBucket() Error: %v", err)
	}

}

type fakeClient struct {
	stiface.Client
	buckets map[string]*fakeBucket
}

type fakeBucket struct {
	attrs   *storage.BucketAttrs
	objects map[string][]byte
}

func newFakeClient() stiface.Client {
	return &fakeClient{buckets: map[string]*fakeBucket{}}
}

func (c *fakeClient) Bucket(name string) stiface.BucketHandle {
	return fakeBucketHandle{c: c, name: name}
}

type fakeBucketHandle struct {
	stiface.BucketHandle
	c    *fakeClient
	name string
}

func (b fakeBucketHandle) Create(_ context.Context, _ string, attrs *storage.BucketAttrs) error {
	if _, ok := b.c.buckets[b.name]; ok {
		return fmt.Errorf("bucket %q already exists", b.name)
	}
	if attrs == nil {
		attrs = &storage.BucketAttrs{}
	}
	attrs.Name = b.name
	b.c.buckets[b.name] = &fakeBucket{attrs: attrs, objects: map[string][]byte{}}
	return nil
}

func (b fakeBucketHandle) Attrs(context.Context) (*storage.BucketAttrs, error) {
	bkt, ok := b.c.buckets[b.name]
	if !ok {
		return nil, fmt.Errorf("bucket %q does not exist", b.name)
	}
	return bkt.attrs, nil
}

func (b fakeBucketHandle) Object(name string) stiface.ObjectHandle {
	return fakeObjectHandle{c: b.c, bucketName: b.name, name: name}
}

type fakeObjectHandle struct {
	stiface.ObjectHandle
	c          *fakeClient
	bucketName string
	name       string
}

func (o fakeObjectHandle) NewReader(context.Context) (stiface.Reader, error) {
	bkt, ok := o.c.buckets[o.bucketName]
	if !ok {
		return nil, fmt.Errorf("bucket %q not found", o.bucketName)
	}
	contents, ok := bkt.objects[o.name]
	if !ok {
		return nil, fmt.Errorf("object %q not found in bucket %q", o.name, o.bucketName)
	}
	return fakeReader{r: bytes.NewReader(contents)}, nil
}

func (o fakeObjectHandle) Delete(context.Context) error {
	bkt, ok := o.c.buckets[o.bucketName]
	if !ok {
		return fmt.Errorf("bucket %q not found", o.bucketName)
	}
	delete(bkt.objects, o.name)
	return nil
}

type fakeReader struct {
	stiface.Reader
	r *bytes.Reader
}

func (r fakeReader) Read(buf []byte) (int, error) {
	return r.r.Read(buf)
}

func (r fakeReader) Close() error {
	return nil
}

func (o fakeObjectHandle) NewWriter(context.Context) stiface.Writer {
	return &fakeWriter{obj: o}
}

type fakeWriter struct {
	stiface.Writer
	obj fakeObjectHandle
	buf bytes.Buffer
}

func (w *fakeWriter) Write(data []byte) (int, error) {
	return w.buf.Write(data)
}

func (w *fakeWriter) Close() error {
	bkt, ok := w.obj.c.buckets[w.obj.bucketName]
	if !ok {
		return fmt.Errorf("bucket %q not found", w.obj.bucketName)
	}
	bkt.objects[w.obj.name] = w.buf.Bytes()
	return nil
}
