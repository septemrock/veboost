# **🤖 Slackbot Usage Flow**

This section outlines the standard procedure for distributing rewards using the Slackbot.

1.  **Convert JSON to CSV**
    -   Use the `convert` command and attach the JSON file containing the voting power data.
    -   The bot will convert the data into a CSV format (`address,amount`).
    -   **Note:** The JSON file can contain data for multiple epochs. The `--epoch` flag specifies which epoch's data to process.
    -   **Command:** `@<bot_name> convert --epoch <epoch_number> --total_amount <amount>`

2.  **Consolidate Unclaimed Rewards**
    -   Use the `export_airdrop` command and attach the CSV file generated in Step 1.
    -   This command first updates the claim status for the previous epoch (`epoch - 1`) by syncing with on-chain data. Then, it integrates the new distribution amounts with any unclaimed rewards from that previous epoch. It outputs a new, consolidated CSV file.
    -   **Note:** The `--epoch` flag represents the new epoch, so it should be set to the `currentEpoch` from the `Airdrop` contract + 1.
    -   **Command:** `@<bot_name> export_airdrop --epoch <epoch_number>`

3.  **Import Data**
    -   Carefully verify the data in the CSV file from Step 2.
    -   Once confirmed, use the `import` command and attach the verified CSV file. This will write the airdrop data into the database for the new epoch and simultaneously create the Merkle tree information for verification.
    -   **Note:** The `--epoch` flag represents the new epoch, so it should be set to the `currentEpoch` from the `Airdrop` contract + 1.
    -   **Command:** `@<bot_name> import --epoch <epoch_number>`

4.  **Verify Data Integrity**
    -   Finally, run the `merkletree` command to perform a final verification of the data integrity in the database.
    -   **Command:** `@<bot_name> merkletree --epoch <epoch_number> --address <address>`

---

### **Checking Airdrop Information**

Use the `airdrop_info` command to retrieve real-time on-chain information about the airdrop contract. This command displays:
- The current airdrop epoch.
- Whether the airdrop is currently active.
- The BR token balance of the airdrop contract.
- The address of the airdrop contract.

This is useful for monitoring the contract's status and balance.

-   **Command:** `@<bot_name> airdrop_info`

---

### **Reverting an Import**

If you import data and realize there was a mistake, you can use the `revert` command. This command will delete the airdrop data and Merkle tree information for the specified epoch, allowing you to import the corrected data.

-   **Important:** You can only revert an epoch that has not yet been set on-chain. The target epoch must be the `currentEpoch` from the contract + 1.
-   **Command:** `@<bot_name> revert --epoch <epoch_number>`
