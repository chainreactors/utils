# jev：以 Claim 向模型求判断

代码提出断言（Claim）并附上证据，Provider 从有限选项中选一个并给出置信度，`Claim.Resolve` 把选择解释为 holds / refuted / insufficient。模型不写回自由文本。内置 `Client` 对接 [TypeSafe Jev](https://docs.typesafe.ai/api.md)（System One），其他模型实现 `Provider` 即可接入。

```go
c, err := jev.NewClient("") // TYPESAFE_API_KEY
p := jev.Cached(c, jev.DefaultCacheSize)
rulings, err := jev.Judge(ctx, p, state, claims)
outcome := claims["k"].Resolve(rulings["k"], jev.DefaultMinConfidence)
```

`jev.Judge` 校验输入，把 state 冻结为 JSON 一次，并校验完整回答；失败的批次不返回任何 Ruling。`DefaultMinConfidence`（0.3）是针对固定模型 `DefaultModel` 在指纹判定验收集上标定的阈值。

## 合约

```go
type Claim struct {
    Statement string
    Options   map[string]Option
}
type Option struct {
    Description string
    Outcome     Outcome // Insufficient / Holds / Refuted
}
type Ruling struct {
    Option     string
    Confidence float64
}
type Provider interface {
    ID() string
    Judge(context.Context, interface{}, map[string]Claim) (map[string]Ruling, error)
}
```

每个 Claim 必须有非空 Statement、非空批次 key，以及显式提供的 `insufficient` 选项，映射到 `Insufficient`。每个 Option 都必须声明合法 Outcome，`jev.Judge` 不自动增补或改写。Provider 返回完整且精确对应的 key 集合，选项必须存在，置信度必须是 [0,1] 内有限值。非法或缺失回答是错误，不当作证据不足。`jev.ValidateClaims` 在调用前校验输入，`jev.ValidateRulings` 在应用或缓存前校验完整批次。

`Outcome` 直接使用字符串 `holds` / `refuted` / `insufficient`，零值为空表示未判定。Claim 的 JSON 为 `statement/options`，Option 为 `description/outcome`，Ruling 为 `option/confidence`；`questions/answers/choice` 只出现在 Jev HTTP 边界，不进入核心和缓存格式。

`claim.Resolve(ruling, minConfidence)` 是唯一的语义解释入口。低置信度解析为 `Insufficient`，但原始 `Ruling.Option` 保留。调用方用代码确定的事实也可以构建合法 Claim/Ruling（置信度 1），经相同的 Resolve 解释。

## 缓存与 Provider

`jev.Cached(p, capacity)` 包装完整批次的精确 LRU，并合并并发相同请求；容量非正时不包装。键包括 Provider.ID、完整提交证据和 Claim 选项及语义，不进行页面相似度复用。失败、非法回答不缓存；返回 map 独立复制。等待者可独立取消，发起者取消会使该次共享请求失败，后续可重试。

`jev.Client` 的 ID 包含 endpoint 和 model，推荐以 `jev.Cached(c, jev.DefaultCacheSize)`（4096）包装。用户可以实现自己的 Provider 包装器处理缓存、限速和计数。Client 只将 Statement 和 Option.Description 转为服务端 choice 格式，不发送本地 Outcome 策略；响应必须显式包含 confidence，缺省或 null 会报错。
