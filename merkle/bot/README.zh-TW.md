# **🤖 Slackbot 使用流程**

本節概述了使用 Slackbot 分發獎勵的標準程序。

1.  **將 JSON 轉換為 CSV**
    -   使用 `convert` 指令並附上包含投票權重數據的 JSON 檔案。
    -   機器人會將數據轉換為 CSV 格式 (`address,amount`)。
    -   **注意：** JSON 檔案可以包含多個週期的數據。`--epoch` 旗標用於指定要處理哪一個週期的數據。
    -   **指令：** `@<bot_name> convert --epoch <epoch_number> --total_amount <amount>`

2.  **整合未領取的獎勵**
    -   使用 `export_airdrop` 指令並附上步驟 1 中產生的 CSV 檔案。
    -   此指令會先透過與鏈上數據同步，來更新前一個週期 (`epoch - 1`) 的領取狀態。然後，它會將新的分發金額與前一個週期未領取的獎勵整合。最後輸岀一個新的、整合後的 CSV 檔案。
    -   **注意：** `--epoch` 旗標代表新的週期，因此應設定為 `Airdrop` 合約中的 `currentEpoch` + 1。
    -   **指令：** `@<bot_name> export_airdrop --epoch <epoch_number>`

3.  **匯入資料**
    -   仔細驗證步驟 2 中 CSV 檔案的數據。
    -   確認無誤後，使用 `import` 指令並附上已驗證的 CSV 檔案。這會將空投數據寫入新週期的資料庫，並同時建立用於驗證的默克爾樹資訊。
    -   **注意：** `--epoch` 旗標代表新的週期，因此應設定為 `Airdrop` 合約中的 `currentEpoch` + 1。
    -   **指令：** `@<bot_name> import --epoch <epoch_number>`

4.  **驗證資料完整性**
    -   最後，執行 `merkletree` 指令，對資料庫中的數據完整性進行最終驗證。
    -   **指令：** `@<bot_name> merkletree --epoch <epoch_number> --address <address>`

---

### **查詢空投資訊**

使用 `airdrop_info` 指令來獲取空投合約的即時鏈上資訊。此指令會顯示：
-   當前的空投週期。
-   空投目前是否處於活躍狀態。
-   空投合約的 BR 代幣餘額。
-   空投合約的地址。

這對於監控合約的狀態和餘額非常有用。

-   **指令：** `@<bot_name> airdrop_info`

---

### **還原匯入操作**

如果您匯入數據後發現有誤，可以使用 `revert` 指令。此指令將刪除指定週期的空投數據和默克爾樹資訊，讓您可以重新匯入修正後的數據。

-   **重要提示：** 您只能還原尚未在鏈上設定的週期。目標週期必須是合約中 `currentEpoch` + 1。
-   **指令：** `@<bot_name> revert --epoch <epoch_number>`
