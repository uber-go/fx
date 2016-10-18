# Kafka consumer

## Sample config HostFile

```yaml
clusters:
   kluster-datacenter1:
     brokers:
       - kluster-datacenter1
       - kluster02-datacenter1
       - kluster03-datacenter1
       - kluster04-datacenter1
       - kluster05-datacenter1
     zookeepers:
       - klusterzk01-datacenter1
       - klusterzk02-datacenter1
     chroot: kluster-datacenter1
```