package com.kernel.sample;

public class PaymentProcessor {
    private final String merchantId;

    public PaymentProcessor(String merchantId) {
        this.merchantId = merchantId;
    }

    public boolean captureFunds(String transactionId, double amount) {
        System.out.printf("Merchant %s captured $%.2f on %s%n", this.merchantId, amount, transactionId);
        return true;
    }
}
