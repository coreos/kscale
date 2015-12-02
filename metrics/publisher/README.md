Metrics Publisher
======

Metrics publisher takes a kubemark density test log (we might extend this in the future), generates reports and uploads them to gcloud storage.

Input is the kubemark log gcloud storage location.

Because we are running nightly jobs, the unit of storage granularity is day. Cloud storage directory layout is defined in [publish_gcloud_storage.sh](./publish_gcloud_storage.sh)

See metrics results in [metrics-kscale](https://console.developers.google.com/storage/browser/metrics-kscale/)
