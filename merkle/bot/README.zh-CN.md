# **🤖 Slackbot 使用流程**

本节概述了使用 Slackbot 分发奖励的标准程序。

1.  **将 JSON 转换为 CSV**
    -   使用 `convert` 命令并附上包含投票权重数据的 JSON 文件。
    -   机器人会将数据转换为 CSV 格式 (`address,amount`)。
    -   **注意：** JSON 文件可以包含多个周期的数据。`--epoch` 标志用于指定要处理哪一个周期的数据。
    -   **命令：** `@<bot_name> convert --epoch <epoch_number> --total_amount <amount>`

2.  **整合未领取的奖励**
    -   使用 `export_airdrop` 命令并附上步骤 1 中生成的 CSV 文件。
    -   此命令会先通过与链上数据同步，来更新前一个周期 (`epoch - 1`) 的领取状态。然后，它会将新的分发金额与前一个周期未领取的奖励整合。最后输出一个新的、整合后的 CSV 文件。
    -   **注意：** `--epoch` 标志代表新的周期，因此应设置为 `Airdrop` 合约中的 `currentEpoch` + 1。
    -   **命令：** `@<bot_name> export_airdrop --epoch <epoch_number>`

3.  **导入数据**
    -   仔细验证步骤 2 中 CSV 文件的数据。
    -   确认无误后，使用 `import` 命令并附上已验证的 CSV 文件。这会将空投数据写入新周期的数据，并同时创建用于验证的默克尔树信息。
    -   **注意：** `--epoch` 标志代表新的周期，因此应设置为 `Airdrop` 合约中的 `currentEpoch` + 1。
    -   **命令：** `@<bot_name> import --epoch <epoch_number>`

4.  **验证数据完整性**
    -   最后，执行 `merkletree` 命令，对数据库中的数据完整性进行最终验证。
    -   **命令：** `@<bot_name> merkletree --epoch <epoch_number> --address <address>`

---

### **查询空投信息**

使用 `airdrop_info` 命令来获取空投合约的即时链上信息。此命令会显示：
-   当前的空投周期。
-   空投目前是否处于活跃状态。
-   空投合约的 BR 代币余额。
-   空投合约的地址。

这对于监控合约的状态和余额非常有用。

-   **命令：** `@<bot_name> airdrop_info`

---

### **还原导入操作**

如果您导入数据后发现有误，可以使用 `revert` 命令。此命令将删除指定周期的空投数据和默克尔树信息，以便您重新导入修正后的数据。

-   **重要提示：** 您只能还原尚未在链上设定的周期。目标周期必须是合约中 `currentEpoch` + 1。
-   **命令：** `@<bot_name> revert --epoch <epoch_number>`
