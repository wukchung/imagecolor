# imagecolor

##### Usage

    ./imagecolor -source=/path/to/source/file -results=/path/to/results/file -cache=/path/to/cache/directory -memlim=512 

##### Description

Main goal is to utilize given HW as much as possible. This doesn't happen out-of-the box and needs to tuned for each HW. Focus is not on algorithm to find the prominent color itself but rather the surrounding. 

###### Feeder
Component fetches data from given list of URLs and store them either in memory or local storage. The assumption is that connectivity is the least reliable resource therefore we need to utilize it as much as possible.

Feeder has following parameters that can be tweaked.

    const (
        maxParallelDownloads   = 10 # maximum amount of downloads running at a time
        maxDataQueueItems      = 100 # maximum amount of items loaded in memory
        maxCacheDataQueueItems = 100000 # maximum amount of items cached on HDD - not implemented
    )

Feeder implements a history which is a duplicate detector but it's limited because we don't have resources to handle input files with more than a billion URLs. This component can be improved a lot by introducing some quick DB but this is not in a scope of this task. It can be turned off by switching `const enableHistory = true`.

###### Image analyzer
Analyze images for the most prominent color. Analyzer can be tweaked by `maxParallelAnalysis` constant. 

###### Limiter
Simple utility to provide a blocking when a limit is reached. The block is released when resource is freed.

###### Memory
Best effort utility to have at least some control over used memory. From a nature of the problem it's almost impossible to do this precisely. We want to make sure to not exhaust all the memory because it would cause application to crash. On the other hand we wan't utilize all the given resources and not be blocked by GC.
 
##### TODO
There are improvements I decided not to do as a part of this this task because I don't think they are in a scope.

- save history to database
- resume functionality to continue from last processed image and use already downloaded files from local storage
- improve logging and error handling
- make all the "tune-up" constants part of configuration
- after reaching local data cache limit pause the downloading
- check size of image before actual download by HEAD request in order to improve memory management
