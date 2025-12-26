/::test_benchmark/ {
  match($0, /::(test_benchmark.*)$/, m)
  test = m[1]
}
/WormholeVerifier::/ {
  match($0, /\[([0-9]+)\]/, g)
  printf "%-50s %10d\n", test, g[1]
}
