start:
	cd openresty && nginx -c conf/nginx.conf

quit:
	cd openresty && nginx -s quit

reload:
	cd ./openresty && nginx -s reload
