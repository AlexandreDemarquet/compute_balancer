import sys

if len(sys.argv) > 1:
    with open("./test.txt", "w") as f:
        f.write(sys.argv[1])
#print(sys.argv[1])
