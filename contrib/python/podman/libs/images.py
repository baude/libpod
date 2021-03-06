"""Models for manipulating images in/to/from storage."""
import collections
import functools
import json

from . import Config
from .containers import Container


class Image(collections.UserDict):
    """Model for an Image."""

    def __init__(self, client, id, data):
        """Construct Image Model."""
        super(Image, self).__init__(data)
        for k, v in data.items():
            setattr(self, k, v)

        self._id = id
        self._client = client

        assert self._id == self.id,\
            'Requested image id({}) does not match store id({})'.format(
                self._id, self.id
            )

    def __getitem__(self, key):
        """Get items from parent dict."""
        return super().__getitem__(key)

    def _split_token(self, values=None, sep='='):
        mapped = {}
        if values:
            for var in values:
                k, v = var.split(sep, 1)
                mapped[k] = v
        return mapped

    def create(self, *args, **kwargs):
        """Create container from image.

        Pulls defaults from image.inspect()
        """
        # Inialize config from parameters
        with self._client() as podman:
            details = self.inspect()

        # TODO: remove network settings once defaults implemented on service side
        config = Config(image_id=self.id, **kwargs)
        config['command'] = details.containerconfig['cmd']
        config['env'] = self._split_token(details.containerconfig['env'])
        config['image'] = details.repotags[0]
        config['labels'] = self._split_token(details.labels)
        config['net_mode'] = 'bridge'
        config['network'] = 'bridge'
        config['work_dir'] = '/tmp'

        with self._client() as podman:
            id = podman.CreateContainer(config)['container']
            cntr = podman.GetContainer(id)
        return Container(self._client, id, cntr['container'])

    container = create

    def export(self, dest, compressed=False):
        """Write image to dest, return id on success."""
        with self._client() as podman:
            results = podman.ExportImage(self.id, dest, compressed)
        return results['image']

    def history(self):
        """Retrieve image history."""
        with self._client() as podman:
            for r in podman.HistoryImage(self.id)['history']:
                yield collections.namedtuple('HistoryDetail', r.keys())(**r)

    def _lower_hook(self):
        """Convert all keys to lowercase."""

        @functools.wraps(self._lower_hook)
        def wrapped(input):
            return {k.lower(): v for (k, v) in input.items()}

        return wrapped

    def inspect(self):
        """Retrieve details about image."""
        with self._client() as podman:
            results = podman.InspectImage(self.id)
        obj = json.loads(results['image'], object_hook=self._lower_hook())
        return collections.namedtuple('ImageInspect', obj.keys())(**obj)

    def push(self, target, tlsverify=False):
        """Copy image to target, return id on success."""
        with self._client() as podman:
            results = podman.PushImage(self.id, target, tlsverify)
        return results['image']

    def remove(self, force=False):
        """Delete image, return id on success.

        force=True, stop any running containers using image.
        """
        with self._client() as podman:
            results = podman.RemoveImage(self.id, force)
        return results['image']

    def tag(self, tag):
        """Tag image."""
        with self._client() as podman:
            results = podman.TagImage(self.id, tag)
        return results['image']


class Images(object):
    """Model for Images collection."""

    def __init__(self, client):
        """Construct model for Images collection."""
        self._client = client

    def list(self):
        """List all images in the libpod image store."""
        with self._client() as podman:
            results = podman.ListImages()
        for img in results['images']:
            yield Image(self._client, img['id'], img)

    def build(self, *args, **kwargs):
        """Build container from image.

        See podman-build.1.md for kwargs details.
        """
        with self._client() as podman:
            # TODO: Need arguments
            podman.BuildImage()

    def delete_unused(self):
        """Delete Images not associated with a container."""
        with self._client() as podman:
            results = podman.DeleteUnusedImages()
        return results['images']

    def import_image(self, source, reference, message=None, changes=None):
        """Read image tarball from source and save in image store."""
        with self._client() as podman:
            results = podman.ImportImage(source, reference, message, changes)
        return results['image']

    def pull(self, source):
        """Copy image from registry to image store."""
        with self._client() as podman:
            results = podman.PullImage(source)
        return results['id']

    def search(self, id, limit=25):
        """Search registries for id."""
        with self._client() as podman:
            results = podman.SearchImage(id, limit)
        for img in results['images']:
            yield collections.namedtuple('ImageSearch', img.keys())(**img)

    def get(self, id):
        """Get Image from id."""
        return next((i for i in self.list() if i.id == id), None)
