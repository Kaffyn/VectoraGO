# TurboQuant Integration in Vectora: Go Implementation Plan

**Documento Técnico** | Status: Design | Data: Abril 2026
**Foco:** Implementação em Go (Microkernel Architecture)

---

## 1. Serialização Binária (TurboQuantFormat)

```go
// Go (aplicável a Go Monolítico ou hybrid)

type CompressedKVEntry struct {
    Version uint8
    Flags   uint8
    KBits   uint8
    VBits   uint8
    KRadius float32
    VRadius float32
    KPackedLen uint16
    VPackedLen uint16
    KSignBitsLen uint16
    VSignBitsLen uint16
    Timestamp int64
    ConvID uint32

    KPacked []uint8       // K quantized bits
    VPacked []uint8       // V quantized bits
    KSignBits []uint8     // QJL 1-bit correction
    VSignBits []uint8     // QJL 1-bit correction

    // Opcional: outliers
    OutlierChannels []uint16 // indices dos canais em 8-bit
    OutlierKValues []float32
    OutlierVValues []float32
}

func (e *CompressedKVEntry) Serialize() ([]byte, error) {
    buf := bytes.NewBuffer(make([]byte, 0, 256))

    // Header
    binary.Write(buf, binary.LittleEndian, e.Version)
    binary.Write(buf, binary.LittleEndian, e.Flags)
    // ... mais campos (KBits, VBits, KRadius, VRadius, Lenns, Timestamp, ConvID)
    binary.Write(buf, binary.LittleEndian, e.KBits)
    binary.Write(buf, binary.LittleEndian, e.VBits)
    binary.Write(buf, binary.LittleEndian, e.KRadius)
    binary.Write(buf, binary.LittleEndian, e.VRadius)
    binary.Write(buf, binary.LittleEndian, e.KPackedLen)
    binary.Write(buf, binary.LittleEndian, e.VPackedLen)
    binary.Write(buf, binary.LittleEndian, e.KSignBitsLen)
    binary.Write(buf, binary.LittleEndian, e.VSignBitsLen)
    binary.Write(buf, binary.LittleEndian, e.Timestamp)
    binary.Write(buf, binary.LittleEndian, e.ConvID)

    // Payloads
    buf.Write(e.KPacked)
    buf.Write(e.VPacked)
    buf.Write(e.KSignBits)
    buf.Write(e.VSignBits)

    // Outliers (se present)
    if e.Flags&0x01 != 0 {
        binary.Write(buf, binary.LittleEndian, uint8(len(e.OutlierChannels)))
        for _, idx := range e.OutlierChannels {
            binary.Write(buf, binary.LittleEndian, idx)
        }
        // ... float32 values (OutlierKValues, OutlierVValues)
        for _, val := range e.OutlierKValues {
            binary.Write(buf, binary.LittleEndian, val)
        }
        for _, val := range e.OutlierVValues {
            binary.Write(buf, binary.LittleEndian, val)
        }
    }

    return buf.Bytes(), nil
}

func Deserialize(data []byte) (*CompressedKVEntry, error) {
    buf := bytes.NewReader(data)
    e := &CompressedKVEntry{}

    binary.Read(buf, binary.LittleEndian, &e.Version)
    binary.Read(buf, binary.LittleEndian, &e.Flags)
    // ... ler Header
    binary.Read(buf, binary.LittleEndian, &e.KBits)
    binary.Read(buf, binary.LittleEndian, &e.VBits)
    binary.Read(buf, binary.LittleEndian, &e.KRadius)
    binary.Read(buf, binary.LittleEndian, &e.VRadius)
    binary.Read(buf, binary.LittleEndian, &e.KPackedLen)
    binary.Read(buf, binary.LittleEndian, &e.VPackedLen)
    binary.Read(buf, binary.LittleEndian, &e.KSignBitsLen)
    binary.Read(buf, binary.LittleEndian, &e.VSignBitsLen)
    binary.Read(buf, binary.LittleEndian, &e.Timestamp)
    binary.Read(buf, binary.LittleEndian, &e.ConvID)

    // Ler payloads
    e.KPacked = make([]uint8, e.KPackedLen)
    buf.Read(e.KPacked)
    e.VPacked = make([]uint8, e.VPackedLen)
    buf.Read(e.VPacked)
    e.KSignBits = make([]uint8, e.KSignBitsLen)
    buf.Read(e.KSignBits)
    e.VSignBits = make([]uint8, e.VSignBitsLen)
    buf.Read(e.VSignBits)

    return e, nil
}
```

---

## 2. TurboQuant Compressor (Go Implementation)

```go
// vectora/pkg/compression/turboquant.go

package compression

import "math"

type TurboQuantCompressor struct {
    HeadDim int
    KBits, VBits uint8
    R RotationMatrix           // (head_dim, head_dim)
    G GaussianMatrix           // (head_dim/2, head_dim)
    KCodebook, VCodebook []float32
}

func NewTurboQuantCompressor(headDim int, kBits, vBits uint8) *TurboQuantCompressor {
    return &TurboQuantCompressor{
        HeadDim: headDim,
        KBits: kBits,
        VBits: vBits,
        R: generateRotationMatrix(42, headDim),
        G: generateGaussianMatrix(43, headDim),
        KCodebook: lloydMaxCodebook(kBits, "beta"),
        VCodebook: lloydMaxCodebook(vBits, "beta"),
    }
}

func (tq *TurboQuantCompressor) CompressKV(k, v [][]float32) (*CompressedKVBatch, error) {
    // k, v: shape [num_heads, head_dim]
    batch := &CompressedKVBatch{
        Heads: make([]*CompressedKVHead, len(k)),
    }

    for headIdx := range k {
        kHead := k[headIdx]
        vHead := v[headIdx]

        // Stage 1: PolarQuant
        kRotated := matmul(tq.R, kHead)
        vRotated := matmul(tq.R, vHead)

        kRadius := norm(kRotated)
        vRadius := norm(vRotated)

        // Normalizar + extract ângulos
        kAngles := extractAngles(normalize(kRotated))
        vAngles := extractAngles(normalize(vRotated))

        // Quantizar
        kIndices := quantize(kAngles, tq.KCodebook)
        vIndices := quantize(vAngles, tq.VCodebook)

        // Pack
        kPacked := packBitstream(kIndices, tq.KBits)
        vPacked := packBitstream(vIndices, tq.VBits)

        // Stage 2: QJL
        kDequant := dequantize(kIndices, tq.KCodebook, kRadius)
        kError := subtract(kDequant, kRotated)
        kErrorProjected := matmul(tq.G, kError)
        kSignBits := toSignBits(kErrorProjected)

        // Análogo para V
        vSignBits := ...

        // Outliers (opcional)
        var outliers *OutlierData
        if shouldStoreOutliers(kRotated) {
            outliers = detectAndStoreOutliers(kRotated, vRotated)
        }

        batch.Heads[headIdx] = &CompressedKVHead{
            HeadIdx:     headIdx,
            KPacked:     kPacked,
            VPacked:     vPacked,
            KSignBits:   kSignBits,
            VSignBits:   vSignBits,
            KRadius:     kRadius,
            VRadius:     vRadius,
            Outliers:    outliers,
        }
    }

    return batch, nil
}

func (tq *TurboQuantCompressor) DecompressKV(batch *CompressedKVBatch) (k, v [][]float32, error) {
    k = make([][]float32, len(batch.Heads))
    v = make([][]float32, len(batch.Heads))

    for headIdx, head := range batch.Heads {
        // Desempacotar
        kIndices := unpackBitstream(head.KPacked, tq.KBits)
        vIndices := unpackBitstream(head.VPacked, tq.VBits)

        // Dequantizar
        kDequant := dequantize(kIndices, tq.KCodebook, head.KRadius)
        vDequant := dequantize(vIndices, tq.VCodebook, head.VRadius)

        // QJL correction
        kSignValues := toSignValues(head.KSignBits)
        // ... reconstruir error_projected approximatively
        kCorrected := add(kDequant, kError)

        // Denormalizar e inverter rotação
        kOriginal := matmul(tq.R.Transpose(), kCorrected)
        vOriginal := matmul(tq.R.Transpose(), vCorrected)

        // Restaurar outliers
        if head.Outliers != nil {
            for _, ch := range head.Outliers.KChannels {
                kOriginal[ch] = head.Outliers.KValues[ch]
            }
        }

        k[headIdx] = kOriginal
        v[headIdx] = vOriginal
    }

    return k, v, nil
}
```

---

## 3. Recuperação e De-serialização

```go
// vectora/pkg/storage/kv_retrieval.go

func (db *KVDatabase) GetCompressedKVForRange(
    conversationID string,
    layer, head int,
    tokenStart, tokenEnd int,
) ([]*CompressedKVEntry, error) {

    key := fmt.Sprintf("kv_cache:layer_%d:head_%d", layer, head)

    entries := []*CompressedKVEntry{}

    // redb range query
    tx := db.readTx()
    defer tx.Close()

    table, err := tx.OpenTable("kv_cache")
    if err != nil {
        return nil, err
    }

    iter, err := table.Range(
        []byte(key + ":" + "token_" + strconv.Itoa(tokenStart)),
        []byte(key + ":" + "token_" + strconv.Itoa(tokenEnd)),
    )
    if err != nil {
        return nil, err
    }

    for iter.Next() {
        keyBuf, valueBuf := iter.KeyValue()

        // Deserialize
        entry, err := deserializeCompressedKV(valueBuf)
        if err != nil {
            log.Warn("deserialize error", "key", keyBuf, "err", err)
            continue
        }

        entries = append(entries, entry)
    }

    return entries, nil
}

func (comp *TurboQuantCompressor) DecompressRange(
    entries []*CompressedKVEntry,
) ([][]float32, [][]float32, error) {

    // Batch decompress múltiplos tokens
    ks := make([][]float32, len(entries))
    vs := make([][]float32, len(entries))

    for i, entry := range entries {
        batch := &CompressedKVBatch{
            Heads: []*CompressedKVHead{
                &CompressedKVHead{
                    KPacked:   entry.KPacked,
                    VPacked:   entry.VPacked,
                    KSignBits: entry.KSignBits,
                    VSignBits: entry.VSignBits,
                    KRadius:   entry.KRadius,
                    VRadius:   entry.VRadius,
                    Outliers:  entry.Outliers,
                },
            },
        }

        k, v, err := comp.DecompressKV(batch)
        if err != nil {
            return nil, nil, err
        }

        ks[i] = k[0]
        vs[i] = v[0]
    }

    return ks, vs, nil
}

// Integration com LLM SDK
func (agent *Agent) CallLLMWithCachedKV(
    conversationID string,
    prompt string,
) (string, error) {

    // 1. Recuperar KV cache do banco
    allLayers := agent.model.NumLayers // ex: 32
    allHeads := agent.model.HeadsPerLayer // ex: 8

    kCache := make([][][]float32, allLayers)
    vCache := make([][][]float32, allLayers)

    for layer := 0; layer < allLayers; layer++ {
        kCache[layer] = make([][]float32, allHeads)
        vCache[layer] = make([][]float32, allHeads)

        for head := 0; head < allHeads; head++ {
            entries, err := agent.kvDB.GetCompressedKVForRange(
                conversationID,
                layer, head,
                0, math.MaxInt32, // todos os tokens até agora
            )
            if err != nil {
                return "", err
            }

            ks, vs, err := agent.compressor.DecompressRange(entries)
            if err != nil {
                return "", err
            }

            // Concatenar todos os tokens deste layer:head
            kCache[layer][head] = concatenate(ks)
            vCache[layer][head] = concatenate(vs)
        }
    }

    // 2. Chamar LLM com KV cache
    // (SDK específico: llama.cpp, genai, etc)
    response, err := agent.llm.GenerateWithKVCache(
        prompt,
        kCache,
        vCache,
    )

    return response, err
}
```

---

## 4. Escrita Incremental e Streaming

```go
// vectora/pkg/storage/kv_write.go

type KVStreamWriter struct {
    db *KVDatabase
    comp *TurboQuantCompressor
    conversationID string
    currentSeqIdx int
    buffer []*CompressedKVEntry
    bufferSize int
    writeInterval time.Duration
}

func (w *KVStreamWriter) WriteToken(
    layer int,
    kLogits [][]float32, // [num_heads, head_dim]
    vLogits [][]float32,
) error {

    // Compressar este token
    batch, err := w.comp.CompressKV(kLogits, vLogits)
    if err != nil {
        return err
    }

    // Serializar para cada head
    for headIdx, headData := range batch.Heads {
        entry := &CompressedKVEntry{
            Version:      0x01,
            KBits:        w.comp.KBits,
            VBits:        w.comp.VBits,
            KPacked:      headData.KPacked,
            VPacked:      headData.VPacked,
            KSignBits:    headData.KSignBits,
            VSignBits:    headData.VSignBits,
            KRadius:      headData.KRadius,
            VRadius:      headData.VRadius,
            Timestamp:    time.Now().Unix(),
            ConversationID: hashConv(w.conversationID),
        }

        w.buffer = append(w.buffer, entry)

        // Flush se buffer cheio
        if len(w.buffer) >= w.bufferSize {
            w.flush(layer, headIdx)
        }
    }

    w.currentSeqIdx++
    return nil
}

func (w *KVStreamWriter) flush(layer, head int) error {
    tx := w.db.writeTx()
    defer tx.Close()

    table, _ := tx.OpenTable("kv_cache")

    for i, entry := range w.buffer {
        key := fmt.Sprintf(
            "kv_cache:layer_%d:head_%d:token_%d:%s",
            layer, head, w.currentSeqIdx-len(w.buffer)+i,
            w.conversationID,
        )

        serialized, _ := entry.Serialize()
        table.Insert([]byte(key), serialized)
    }

    tx.Commit()
    w.buffer = w.buffer[:0]

    return nil
}

// Streaming flush (durante generation)
func (agent *Agent) StreamGenerate(
    conversationID string,
    prompt string,
    onToken func(token string),
) error {

    kvWriter := &KVStreamWriter{
        db:            agent.kvDB,
        comp:          agent.compressor,
        conversationID: conversationID,
        bufferSize:    32, // flush a cada 32 tokens
        writeInterval: 500 * time.Millisecond,
    }

    // Chamar LLM com callbacks
    err := agent.llm.StreamGenerateWithCallbacks(
        prompt,
        func(token string, kv *KVActivations) error {
            // KV ativações deste token
            for layer := 0; layer < agent.model.NumLayers; layer++ {
                k := kv.K[layer]
                v := kv.V[layer]
                kvWriter.WriteToken(layer, k, v)
            }

            onToken(token)
            return nil
        },
    )

    kvWriter.flush(0, 0) // final flush
    return err
}
```

---

## 5. TTL e Política de Expulsão (Eviction)

```go
// vectora/pkg/storage/kv_eviction.go

type TTLManager struct {
    db *KVDatabase
    tickInterval time.Duration
}

func (ttl *TTLManager) EvictExpired(conversationID string) error {
    now := time.Now().Unix()

    tx := ttl.db.writeTx()
    defer tx.Close()

    ttlTable, _ := tx.OpenTable("cache:ttl")

    // Scan and remove expired entries
    iter, _ := ttlTable.Iter()
    toDelete := [][]byte{}

    for iter.Next() {
        key, valBuf := iter.KeyValue()

        var ttlEntry TTLEntry
        json.Unmarshal(valBuf, &ttlEntry)

        if ttlEntry.ExpiresAt < now {
            toDelete = append(toDelete, key)
        }
    }

    for _, key := range toDelete {
        ttlTable.Remove(key)
    }

    tx.Commit()
    return nil
}

// Política: LRU com priority
func (ttl *TTLManager) SetTTL(
    conversationID string,
    tokenSeq int,
    expiresAt int64,
    priority uint8,
) error {

    key := fmt.Sprintf("cache:ttl:%s:token_%d", conversationID, tokenSeq)

    entry := TTLEntry{
        ExpiresAt: expiresAt,
        Priority:  priority,
    }

    serialized, _ := json.Marshal(entry)

    tx := ttl.db.writeTx()
    defer tx.Close()

    table, _ := tx.OpenTable("cache:ttl")
    table.Insert([]byte(key), serialized)

    return tx.Commit()
}

// Eviction automático (background worker)
func (ttl *TTLManager) Start(ctx context.Context) {
    ticker := time.NewTicker(5 * time.Minute)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            ttl.EvictExpired("")
        }
    }
}
```

---

## 6. Integração llama.cpp (Hook Layer)

```go
// Implementação em Go (via cgo ou rust FFI)
//export llama_kv_callback_impl
func llama_kv_callback_impl(token *C.llama_kv_token, userData unsafe.Pointer) C.int {
    // Converter para Go types
    layer := int(token.layer)
    head := int(token.head)

    kHead := (*[128]float32)(unsafe.Pointer(token.k_data))[:]
    vHead := (*[128]float32)(unsafe.Pointer(token.v_data))[:]

    // Comprimir e armazenar
    compressor := (*TurboQuantCompressor)(userData)
    // Nota: CompressKV requer []float32 p/ head
    batch, _ := compressor.CompressKV([][]float32{kHead}, [][]float32{vHead})

    // Escrever para DB via writer injetado
    kvWriter.WriteToken(layer, [][]float32{kHead}, [][]float32{vHead})

    return 0 // success
}

// Uso em Go:
func runLlamaWithStorage(model string, prompt string) {
    ctx, _ := llama.New(model)
    defer ctx.Close()

    compressor := NewTurboQuantCompressor(128, 3, 4)
    kvWriter := NewKVStreamWriter(db, compressor, "conv_id")

    // Setup callback
    ctx.SetKVCallback(func(layer, head int, k, v []float32) {
        kvWriter.WriteToken(layer, [][]float32{k}, [][]float32{v})
    })

    // Generate (KV é capturado automaticamente)
    tokens := ctx.Generate(prompt, 100)
    fmt.Println(llama.Decode(tokens))
}
```
