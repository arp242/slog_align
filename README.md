Less "wall of text"-y slog handler; mainly for dev/testing.

I've seen so many apps that will just spit walls and walls of text on error
failures and trying to get something meaningful out of it is like manually
digging through War & Peace to find that one 6-word quote.

I don't know why all these go loggers have such terrible output by default; Just
printing things a bit aligned makes such a world of difference. I must've
written a handler like this 5 or 6 times for different loggers over the years by
now.

Default handler:

    2024/03/23 07:21:02 ERROR error
    2024/03/23 07:21:02 WARN warn
    2024/03/23 07:21:02 INFO info
    2024/03/23 07:21:02 ERROR error str=foo int=123 map="map[another:true key:value struct:{s:[1 2 3]}]" slice="[a b]" struct="{s:asd i:123}"
    2024/03/23 07:21:02 WARN warn str=foo int=123 map="map[another:true key:value struct:{s:[1 2 3]}]" slice="[a b]" struct="{s:asd i:123}"
    2024/03/23 07:21:02 INFO info str=foo int=123 map="map[another:true key:value struct:{s:[1 2 3]}]" slice="[a b]" struct="{s:asd i:123}"

slog_align:

    07:25 ERROR error                                                                       main/main.go:13
    07:25 WARN  warn                                                                        main/main.go:14
    07:25 INFO  info                                                                        main/main.go:15
    07:25 DEBUG debug                                                                       main/main.go:16
    07:25 ERROR error                                                                       main/main.go:32
                str    = foo
                int    = 123
                map    = map[another:true key:value struct:{[1 2 3]}]
                slice  = [a b]
                struct = {asd 123}
    07:25 WARN  warn                                                                        main/main.go:33
                str    = foo
                int    = 123
                map    = map[another:true key:value struct:{[1 2 3]}]
                slice  = [a b]
                struct = {asd 123}
    07:25 INFO  info                                                                        main/main.go:34
                str    = foo
                int    = 123
                map    = map[another:true key:value struct:{[1 2 3]}]
                slice  = [a b]
                struct = {asd 123}
    07:25 DEBUG debug                                                                       main/main.go:35
                str    = foo
                int    = 123
                map    = map[another:true key:value struct:{[1 2 3]}]
                slice  = [a b]
                struct = {asd 123}


---

Use it as:

	l := slog.New(NewAlignedHandler(os.Stdout))
	l.Info("yo yo")


Or globally:

	slog.SetDefault(slog.New(slog_align.NewAlignedHandler(os.Stdout)))
	slog.Info("yo yo")
