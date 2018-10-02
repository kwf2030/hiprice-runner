# hiprice-runner
Crawler for HiPrice.

## Run
- hiprice-runner uses Chrome/Chromium(64+) for crawling. Make sure you have installed.
- hiprice-runner uses Beanstalk as job queue, Make sure you have installed.
- Validate conf.yaml(Beanstalk host/port and Chrome exec/args).
- Compile this repo with `go build`, close all Chrome/Chromium instances and execute the binary directly.