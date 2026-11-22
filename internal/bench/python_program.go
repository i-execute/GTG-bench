package bench

const pythonProgram = `import base64
import importlib.metadata
import json
import sys
import time

LIBRARIES = {}
ERRORS = {}
try:
    from goygram import ext as goygram_ext
    LIBRARIES["GoyGram"] = (goygram_ext.aes_ige_enc, goygram_ext.aes_ige_dec, importlib.metadata.version("GoyGram"))
except Exception as error:
    ERRORS["GoyGram"] = str(error)
try:
    import tgcrypto
    LIBRARIES["TgCrypto"] = (tgcrypto.ige256_encrypt, tgcrypto.ige256_decrypt, importlib.metadata.version("TgCrypto"))
except Exception as error:
    ERRORS["TgCrypto"] = str(error)
try:
    import cryptg
    LIBRARIES["cryptg"] = (cryptg.encrypt_ige, cryptg.decrypt_ige, importlib.metadata.version("cryptg"))
except Exception as error:
    ERRORS["cryptg"] = str(error)

def pattern(length, salt):
    return bytes((index * 131 + salt) & 255 for index in range(length))

def library(name):
    if name not in LIBRARIES:
        raise RuntimeError(ERRORS.get(name, "not installed"))
    return LIBRARIES[name]

def evict(size):
    block = bytearray(max(8 * 1024 * 1024, min(128 * 1024 * 1024, size * 2)))
    value = 0
    for index in range(0, len(block), 64):
        block[index] = (index * 17 + 3) & 255
        value ^= block[index]
    return value

def qualify(request):
    name = request["library"]
    if name not in LIBRARIES:
        return {"ok": False, "available": False, "reason": ERRORS.get(name, "not installed")}
    encrypt, decrypt, version = library(name)
    plain = pattern(request.get("size", 64), 31)
    key = pattern(32, 11)
    iv = pattern(32, 17)
    try:
        ciphertext = encrypt(plain, key, iv)
        opened = decrypt(ciphertext, key, iv)
        return {"ok": opened == plain, "available": True, "version": version, "ciphertext": base64.b64encode(ciphertext).decode(), "plaintext": base64.b64encode(opened).decode(), "reason": "" if opened == plain else "reverse decryption mismatch"}
    except Exception as error:
        return {"ok": False, "available": True, "version": version, "reason": str(error)}

def measure(request):
    name = request["library"]
    if name not in LIBRARIES:
        return {"ok": False, "available": False, "reason": ERRORS.get(name, "not installed")}
    encrypt, decrypt, version = library(name)
    size = int(request["size"])
    iterations = max(1, int(request["iterations"]))
    plain = pattern(size, 31)
    key = pattern(32, 11)
    iv = pattern(32, 17)
    try:
        input_data = plain if request["direction"] == "encrypt" else encrypt(plain, key, iv)
        if request.get("cold", False):
            cold_value = evict(size)
        else:
            cold_value = 0
        fn = encrypt if request["direction"] == "encrypt" else decrypt
        checksum = cold_value
        start = time.perf_counter_ns()
        for _ in range(iterations):
            output = fn(input_data, key, iv)
            checksum ^= output[0] << 8 | output[-1]
        elapsed = time.perf_counter_ns() - start
        return {"ok": True, "available": True, "version": version, "elapsed_ns": elapsed, "checksum": checksum}
    except Exception as error:
        return {"ok": False, "available": True, "version": version, "reason": str(error)}

def main():
    for line in sys.stdin:
        try:
            request = json.loads(line)
            if request.get("action") == "qualify":
                response = qualify(request)
            elif request.get("action") == "measure":
                response = measure(request)
            else:
                response = {"ok": False, "reason": "unknown action"}
        except Exception as error:
            response = {"ok": False, "reason": str(error)}
        sys.stdout.write(json.dumps(response, separators=(",", ":")) + "\n")
        sys.stdout.flush()

main()
`
