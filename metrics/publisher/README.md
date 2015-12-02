Metrics Publisher
======

Metrics publisher takes a kubemark density test log (we might extend this in the future), generates reports and uploads them to gcloud storage.

Input is the kubemark log gcloud storage location.

Because we are running nightly jobs, the unit of storage granularity is day. Cloud storage directory layout for one publication will look like:
```sh
{$bucket_name}/kubemark/{$date: 2015-11-27}/
```

See metrics results in [metrics-kscale](https://console.developers.google.com/storage/browser/metrics-kscale/)