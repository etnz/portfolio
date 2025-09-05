## Cost Basis Methods

When you invest, understanding how capital gains are calculated is crucial, especially for tax purposes. A **capital gain** is the profit you make from selling an asset (like a stock or a fund) for more than you bought it. Conversely, a **capital loss** occurs when you sell an asset for less than its purchase price.

There are two main types of capital gains:

*   **Realized Gains/Losses**: These occur when you actually sell an asset. Realized gains are typically subject to taxation.
*   **Unrealized Gains/Losses**: These are the theoretical profits or losses on assets you still hold. They only become realized (and taxable) when you sell the asset.

To calculate a realized capital gain or loss, you need to determine the **cost basis** of the asset sold. The cost basis is essentially the original value of an asset for tax purposes, usually its purchase price. However, when you buy the same security multiple times at different prices, determining which "lot" was sold can be complex. This is where different cost basis methods come into play.

Many countries have specific rules or even legally binding methods for calculating the cost basis of securities for tax purposes. It's important to be aware of these regulations in your jurisdiction. For a broader overview of how different countries approach capital gains taxation, you can refer to resources like the [OECD's report on Taxing Capital Gains](https://www.oecd.org/content/dam/oecd/en/publications/reports/2025/02/taxing-capital-gains_76a32327/9e33bd2b-en.pdf).

`pcs` supports two common cost basis methods for calculating capital gains:

*   **`average`**: This method calculates the cost basis by averaging the cost of all shares. For example, in France, the *Prix Moyen Pondéré d'Acquisition* (PMP), which is a weighted average cost method, is legally binding for individual capital gains on fungible securities. For more details, refer to the official French tax documentation: [impots.gouv.fr](https://www.impots.gouv.fr/portail/particulier/plus-values-de-cessions-de-valeurs-mobilieres-et-droits-sociaux).
*   **`fifo`**: This method (First-In, First-Out) assumes that the first shares purchased are the first ones sold. For instance, in Germany, the FIFO principle is legally binding for capital gains tax purposes on securities. For more details, refer to relevant German tax information, e.g., [fondsvermittlung24.de](https://www.fondsvermittlung24.de/abgeltungsteuer-fifo-methode/).



