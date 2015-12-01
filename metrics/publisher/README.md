Metrics Publisher
======

Metrics publisher takes all reports from a directory and uploads them to cloud storage.

Input is just a directory.
Publisher will takes only the files (reports) within.
It doesn't upload the directory because publisher needs to create a new one according to time format.

Because we are running nightly jobs, the unit of storage granularity is day. Cloud storage directory layout for one publication will look like:
```sh
{$bucket_name}/kubemark/{$date: 2015-11-27}/
```

See metrics results in [metrics-kscale](https://console.developers.google.com/storage/browser/metrics-kscale/)