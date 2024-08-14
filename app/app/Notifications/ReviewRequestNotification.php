<?php

namespace App\Notifications;

use App\Models\ReviewRequest;
use Illuminate\Bus\Queueable;
use Illuminate\Contracts\Queue\ShouldQueue;
use Illuminate\Notifications\Messages\MailMessage;
use Illuminate\Notifications\Notification;

class ReviewRequestNotification extends Notification
{
    use Queueable;

    protected ReviewRequest $reviewRequest;
    /**
     * Create a new notification instance.
     */
    public function __construct(ReviewRequest $reviewRequest)
    {
        //
        $this->reviewRequest = $reviewRequest;
    }

    /**
     * Get the notification's delivery channels.
     *
     * @return array<int, string>
     */
    public function via(object $notifiable): array
    {
        return ['mail'];
    }

    /**
     * Get the mail representation of the notification.
     */
    public function toMail(object $notifiable): MailMessage
    {
        return (new MailMessage())
            ->greeting("Hi {$notifiable->name}")
            ->subject('Review Request')
            ->line("{$this->reviewRequest->requester->name} has requested a review from you.")
            ->action('Review', url('/'));
    }

    /**
     * Get the array representation of the notification.
     *
     * @return array<string, mixed>
     */
    public function toArray(object $notifiable): array
    {
        return [
                //
            ];
    }
}
