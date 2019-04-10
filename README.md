# Metrics

## Why

Tired of looking for a package that can do performance counters and events in statsd or something else I decided to write metrics.

Metrics doesn't try to be the fastest, rather it tires to be useable.

## Feature list

Metric types

- [x] Counters
- [x] Gauges
- [x] Timers
- [x] Polymorphic metrics
- [ ] ~~Histograms~~ Maybe later...

Features on metrics

- [x] Tagging on individual metrics
- [x] Sample rate for counters
- [x] Shipper with plugins
- [ ] Shipper: UDP/TCP statsd
- [ ] Shipper: Influx HTTP(S)
- [x] Shipper: STDOUT
- [x] Shipper: DevNull
- [x] Default tagging all metrics
- [x] Serialization support for metrics as plugins
- [ ] Serialization: Influx Line protocol
- [ ] Serialization: Statsd with datadog and influx tagging
- [x] Serialization: JSON with pretty printing

## What is a Metric

Ahhh, Wikipedia please: [Software Metric](https://en.wikipedia.org/wiki/Software_metric)

*Extract below*

```text
A software metric is a standard of measure of a degree to which a software system or process possesses some property. Even if a metric is not a measurement (metrics are functions, while measurements are the numbers obtained by the application of metrics), often the two terms are used as synonyms. Since quantitative measurements are essential in all sciences, there is a continuous effort by computer science practitioners and theoreticians to bring similar approaches to software development. The goal is obtaining objective, reproducible and quantifiable measurements, which may have numerous valuable applications in schedule and budget planning, cost estimation, quality assurance, testing, software debugging, software performance optimization, and optimal personnel task assignments.
```

To this package a metric is really just a measurement with some metadata attached to it or an event with some metadata attached to it.

For simplicity I will refer to measurements as metrics in the rest of this document.

## Why would I want to put metrics in my application

To tell you whats happening inside your application.

Its really that simple. Measuring and monitoring your application is just something that you will need to do as it grows.

## Shipping and Serialization of metrics

### Shippers

Metrics gathered are only useful if they go somewhere. Thats what shippers are for. They take the metrics serialize them to fit the data structure of the receiver and then send them.

### Serialization

The receiving end of the metric needs to know how to ready it. So we can serialize the message into something they would be able to read. Most metric gathering platforms have the same features, metrics with fields, tags and time stamps. How that data is presented is normally the difference.

The metrics don't come with timestamps as many receivers will put the time stamps on it for you. However there is nothing stopping the serializer from adding a time stamp as it knows what it should be and where in the serialized data format.