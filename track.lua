local sk, su, ss, of, ofs, ofe, vo, rfsc, ex

ex = ARGV[4]

sk = KEYS[2]
su = ARGV[2]
rfsc = KEYS[3]
if su == "@" then
  su = redis.call("HGET", rfsc, ARGV[5])
else
  redis.call("HSET", rfsc, ARGV[5], su)
end
redis.call("EXPIRE", rfsc, ex)

ss = redis.call("HGET", sk, su) or "~"

of = tonumber(ARGV[3])
if of < 0 then of = (string.len(ss)-1)/2 end

ofs = of*2+2
ofe = ofs+1

vo = string.sub(ss, ofs, ofe)
if vo == "" then vo = 0 else vo = tonumber(vo) end

vo = vo+tonumber(ARGV[6])
if vo > 99 then vo = 99 end
vo = tostring(vo)
if string.len(vo) == 1 then
  vo = "0" .. vo
end
ss = string.sub(ss, 0, ofs-1) .. vo .. string.sub(ss, ofe+1)
redis.call("HSET", sk, su, ss)

redis.call("EXPIRE", sk, ex)

redis.call("HINCRBY", KEYS[1], ARGV[1], 1)
redis.call("EXPIRE", KEYS[1], ex)

return of
