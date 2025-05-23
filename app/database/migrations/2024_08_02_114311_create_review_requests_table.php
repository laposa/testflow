<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration {
    /**
     * Run the migrations.
     */
    public function up(): void
    {
        Schema::create('review_requests', function (Blueprint $table) {
            $table->id();
            $table
                ->foreignId('session_id')
                ->constrained('test_sessions')
                ->onDelete('cascade')
                ->onUpdate('cascade');
            $table->foreignId('requester_id')->constrained('users');
            $table->foreignId('reviewer_id')->constrained('users');
            $table->string('status')->default('pending');
            $table->dateTime('completed_at')->nullable();
            $table->timestamps();
        });
    }

    /**
     * Reverse the migrations.
     */
    public function down(): void
    {
        Schema::dropIfExists('review_requests');
    }
};
