# Product Value Proposition: pcs

## Vision

To be the permanent and unified home for your entire financial life, making wealth tracking effortless and insightful over decades.

## Core Value Proposition

For the long-term investor with a scattered and diverse portfolio, `pcs` is the **durable** command-line tool that **unifies** your entire financial world.

Unlike cloud-based platforms that can disappear and force you into a painful migration from a poorly documented format that risks derailing your tracking efforts entirely, `pcs` provides a permanent, auditable, and easily-maintainable record of your wealth. It is **built to last**.

It makes it effortless to answer the two most critical questions at any point in your financial journey: "What is my net worth?" and "How is it performing?"**. It provides **essentials clarity and insights**.

## The "Why": Key Pillars of `pcs`

### Pillar 1: Unify Everything

The modern investor's wealth is fragmented. It lives in corporate savings plans with private funds, in multiple brokerage accounts, in stock option portals, and in physical assets like real estate. This fragmentation makes a single view of one's net worth nearly impossible to achieve.

* **`pcs` solves this by being asset-agnostic.** It is designed from the ground up to track a diverse range of assets:

  * Publicly traded stocks and ETFs.

  * Private funds, frequent in corporate savings plans and retirement plans.

  * Liabilities and counterparty accounts (e.g., taxes owed, loans).

  * Future support for stock options, real estate, and more.

The primary goal is to consolidate every part of your wealth into one coherent picture.

### Pillar 2: Built to Last (Durability & Effortless Tracking)

Wealth tracking is a multi-decade endeavor. The biggest barrier to long-term tracking is friction; if the process is too cumbersome, it won't be sustained. `pcs` is built for durability by making the process of maintaining your financial record as seamless as possible.

* **`pcs` is built for durability, giving you ultimate control:**

  * **Local-First:** Your data lives in a simple text file (`transactions.jsonl`) on your machine. You own it, forever.

  * **Auditable & Versionable:** The text-based format works perfectly with Git. This creates an immutable, auditable history of your wealth.

  * **Easy Backtracking:** Starting a portfolio today means recording years of history. `pcs` is designed to make this backtracking process as simple as possible, allowing you to easily add, correct, and annotate past transactions.

  * **Frictionless Data Entry:** The primary goal is to make tracking sustainable. The **extension mechanism** allows for connecting to any data source, and future integrations with **AI-powered importers** (e.g., reading transaction data from screenshots) will further streamline this process.

### Pillar 3: Essential Clarity & Insight

Financial tools often overwhelm users with complex analytics that don't answer their core questions. `pcs` focuses on providing essential clarity.

* **`pcs` focuses on simplicity and clarity:**

  * **Answering the Core Questions:** The analysis provided by `pcs` is intentionally focused on the two questions that matter most: "What is my net worth?" and "How is it performing?".


## Future Directions and Ideas

This section outlines potential future directions for `pcs`, framed as solutions to common, long-term investor problems. These are not committed features but high-level ideas that explore how the tool could evolve in alignment with our core product value proposition. They serve as a basis for community discussion and future prioritization.

### Pillar 1: Unify Everything

* **1. The Problem of "Potential Wealth":** A significant part of my compensation is tied up in employee stock options, but they aren't part of my net worth until they vest and I exercise them. It's difficult to track this "potential" wealth alongside my actual assets without distorting my current financial picture. I need a way to see both my current net worth and the future potential from these grants in one place.

* **2. The Problem of Infrequent Valuation for Illiquid Assets:** My biggest assets, like my house or private funds, don't have a daily market price. This makes my net worth calculation static and inaccurate for long periods. I get a precise value when I buy or sell, but in between, I'm flying blind. I need a way to get a *reasonable estimate* of their value on any given day, using public data like market indexes as a proxy, and be able to correct this estimate whenever I get a new 'real' valuation, ensuring my historical net worth is always as accurate as possible.

* **3. The Problem of "Financial Compartments":** My personal finances are separate from the joint accounts I share with my family, and I also manage a small portfolio for my children. I need the flexibility to analyze these "financial compartments" individually while also being able to see a consolidated, aggregated view of our entire family's wealth.

### Pillar 2: Built to Last (Durability & Effortless Tracking)

* **4. The Problem of Tedious Data Entry:** Manually typing every transaction from my bank's web interface or PDF statements is the biggest chore in wealth tracking. It's so time-consuming and error-prone that it makes me want to give up. I need a radically faster and easier way to get my transaction history into `pcs`.

* **5. The Problem of Correcting the Past:** When I started my portfolio, I had to add years of historical data, and I know I made mistakes. Finding and fixing a single incorrect entry from five years ago in a long text file is intimidating and I'm afraid I'll break something. I need a safe and intuitive way to edit my financial history without risk.

* **6. The Problem of "Hidden" Liabilities:** Every time I sell a stock for a profit, I create a future tax liability that isn't immediately visible. My net worth looks higher than it actually is because I forget to account for the money I'll eventually owe the government. I need a way to automatically track the tax implications of my actions as they happen.

* **7. The Problem of Data "Rot":** Over many years, small errors can creep into my financial record—a typo in a price, a transaction I forgot to categorize. These small issues can compound and make me lose trust in my own data. I need a way to periodically audit my ledger to find and fix these inconsistencies, ensuring my financial record remains accurate and reliable for decades to come.

### Pillar 3: Essential Clarity & Insight

* **8. The Problem of "Context-Free" Numbers:** My report tells me I'm up or down by a certain percentage, but I have no idea *why*. Was it because one stock did particularly well, or was it a broader market trend? I need a narrative that explains the story behind the numbers and connects my portfolio's performance to real-world events.

* **9. The Problem of "Model-Driven Uncertainty":** My portfolio's value, both past and future, relies on predictive models—from index-based estimators for my illiquid assets to market trend projections for scenario planning. These models are powerful, but they are also assumptions. I have no way to validate how well my chosen estimators have performed against real data in the past, nor can I easily see how sensitive my actual wealth estimation is to changes in these assumptions. I need a way to backtest my models' historical accuracy and perform sensitivity analysis to understand the range of potential outcomes, giving me confidence not just in the numbers, but in the system I'm using to generate them.

* **10. The Problem of an Uncertain Future:** My portfolio is a perfect record of the past, but it tells me nothing about the future. My financial future is a complex mix of predictable events (like mortgage payments and stock vesting dates) and unpredictable ones (like market crashes, when I might buy a house, or a potential inheritance). I need more than a simple forecast; I need a tool that lets me model different future **scenarios** so I can understand the potential range of my future wealth and make better long-term decisions. What happens to my 10-year outlook if the market crashes next year and takes two years to recover? What if I inherit later than expected?
