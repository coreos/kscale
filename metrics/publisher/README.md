Metrics Publisher
======

Metrics publisher takes all reports from a directory and uploads them to cloud storage.

Input is just a directory.
Publisher will takes only the files (reports) within.
It doesn't upload the directory because publisher needs to create a new one according to time format.

Because we are running nightly jobs, the unit of storage granularity is day. Cloud storage directory layout for one publication will look like:
```sh
kscale/results/kubemark/{$date: 2015-11-27}/
```