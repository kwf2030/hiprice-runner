# hiprice-runner
Crawler for HiPrice.

## Run
- hiprice-runner uses Chrome/Chromium(64+) for crawling. Make sure you have installed.
- hiprice-runner uses Beanstalk as job queue, Make sure you have installed.
- Check Beanstalk host/port and Chrome exec/args in conf.yaml.
- Close all Chrome/Chromium instances.
- Compile this repo with `go build`, execute the binary directly.
