FROM gliderlabs/alpine

ADD https://raw.githubusercontent.com/antirez/redis/3.0/src/redis-trib.rb /usr/local/bin/redis-trib.rb

RUN apk --update add ruby redis && gem install redis --no-doc && chmod +x /usr/local/bin/redis-trib.rb
