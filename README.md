# Visa and mastercard

This draft document is for writing all necessary steps to integrate Visa and MasterCard network in Sudan. It intended to enlist all road blocks and all necessary requirements so we can anticipate them.


## What are we building

Noebs is an open source payment gateway that connects to EBS channels through Consumer and Merchant. Noebs is highly performant, secure payment gateway and is actively maintained. 

### How noebs works

_or payment primiere_

Noebs is connected to EBS via an HTTP / RESTful apis for both consumer and merchant. EBS merchant and consumer are a wraper built on top of their TWO (TanzWare Online Switch System). 

Noebs and likewise EBS is also only Magnetic Card (MagCard) compatible, they don't use nor support EMV based transactions. MagCard are very easy and simple to implement, essentially what we do is just sending PAN, PIN, expDate. The card in itself doesn't provide any mechanism for security. 

EMV has built-in security features that makes it impossible for counterfeit frauds. In MagCard, we cannot guarantee a transaction originated from the genunie card--we can only hope for that. The security in the MagCard is in:
    - PAN
    - PIN
    - ExpDate

PAN and expdate are printed in the card itself. They are easily copy-able. PIN is not very hard to guess either. And for certain kind of transactions (not currently available in Sudan), PIN is not even required. For example consider transactions below 100 SDG-- it is not seamless to ask a card holder to enter their PIN for such pity transactions. Another point is that such transactions can posses a larger risk for the card holder *if* they were to enter their PIN.

EMV with its built in security allows for more secure transactions. EMV can support many more transactions, for example:

- EMV provides mechansim against card cloning by means of issuing certified keys for card issuers
- offline transactions is permitted within EMV framework
- In EMV transaction scope, a terminal can apply certain security and risk management steps to further secure transactions
- Card holder chip (EMV chip) can provides a list of preferred payment schemes (For example PIN only, PIN + otp, CVV only, and so forth). This powerful configurability is a key drive for EMV adoption.
- EMV differentiates card issuers by using an application identifier. For example Visa cards have different AID than mastercard. This is both a convenient feature and a source of bugs. The reason for that is different card issuers have different EMV payment flows and it is left for the POS terminal to support / not support any of them. There's no general framework to accept every EMV card issuer and it is the responsibility of the terminal provider to choose what they will be supporting.


## What are we supporting

Having said that, it becomes vital to know what kind of cards we are supporting. Omni-payment channel is never an easy exercise to do. Especially when working with EMV payment flows, each card issuer has their own integration schemes. 

The proposed solution must also be PCI compliant. While integrating with Visa / mastercard through Magnetic Stripe, it is less secure to do that. Fully supporting Visa and MasterCard is a huge endeavor and a potentially great way of differentiating our product from any other competitor. Ultimately, EMV chip integration is the industry standard when processing payments. 